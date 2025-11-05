package upstream

import (
	"context"
	"net/http"

	"gcli2api-go/internal/credential"
	route "gcli2api-go/internal/upstream/strategy"
)

// RotationOptions controls in-request credential rotation behaviour.
type RotationOptions struct {
	// MaxRotations caps alternate credential switches within a single request.
	// If <=0, a safe default is computed from available credentials (up to 8).
	MaxRotations int
	// RotateOn5xx toggles rotation on 5xx responses.
	RotateOn5xx bool
}

// TryWithRotation executes do(cred) and, on certain status codes, rotates credentials
// using credMgr and router up to MaxRotations. It returns the final response (not closed),
// the credential used for that response, and error (if any).
func TryWithRotation(
	ctx context.Context,
	credMgr *credential.Manager,
	router *route.Strategy,
	initial *credential.Credential,
	opts RotationOptions,
	do func(c *credential.Credential) (*http.Response, error),
) (*http.Response, *credential.Credential, error) {
	current := initial
	// Compute default cap when unset
	maxRot := opts.MaxRotations
	if maxRot <= 0 && credMgr != nil {
		creds := credMgr.GetAllCredentials()
		if n := len(creds); n > 0 {
			maxRot = n * 2
			if maxRot > 8 {
				maxRot = 8
			}
			if maxRot < 2 {
				maxRot = 2
			}
		} else {
			maxRot = 4
		}
	} else if maxRot <= 0 {
		maxRot = 4
	}

	rotations := 0
	for {
		release := func() {}
		if current != nil && credMgr != nil {
			release = credMgr.Acquire(current.ID)
		}
		resp, err := do(current)
		// capture status code for decisions
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		release()

		// success path
		if err == nil && resp != nil && status < 400 {
			return resp, current, nil
		}

		// rotation/refresh decisions only if we have a credential manager
		if resp != nil && current != nil && credMgr != nil {
			code := resp.StatusCode
			// 401: try compensating refresh via router once
			if code == http.StatusUnauthorized && router != nil {
				if fresh, ok := router.Compensate401(ctx, current.ID); ok && fresh != nil {
					// retry once with refreshed credential
					current = fresh
					// do not count as a rotation yet
					resp2, err2 := do(current)
					status2 := 0
					if resp2 != nil {
						status2 = resp2.StatusCode
					}
					if err2 == nil && resp2 != nil && status2 < 400 {
						return resp2, current, nil
					}
					// fallback to rotation checks using resp2/err2
					if resp2 != nil {
						resp = resp2
						err = err2
						code = status2
					}
				}
			}
			// Rotation on known retryable statuses
			rotate := code == 429 || code == 401 || code == 403 || (opts.RotateOn5xx && code >= 500 && code <= 599)
			if rotate {
				// record failure for current credential
				credMgr.MarkFailure(current.ID, "upstream_error", code)
				if router != nil {
					router.OnResult(current.ID, code)
				}
				if alt, errAlt := credMgr.GetAlternateCredential(current.ID); errAlt == nil && alt != nil {
					rotations++
					if rotations >= maxRot {
						// return the last response (do not close here)
						return resp, current, err
					}
					// We are going to try another credential; close the previous response body
					if resp != nil {
						_ = resp.Body.Close()
					}
					current = alt
					continue
				}
			}
		}
		// No rotation happened; return the last response as-is
		return resp, current, err
	}
}
