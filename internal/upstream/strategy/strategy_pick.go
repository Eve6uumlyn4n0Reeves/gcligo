package strategy

import (
	"context"
	"net/http"
	"time"

	"gcli2api-go/internal/credential"
	mon "gcli2api-go/internal/monitoring"
)

// Pick 选取一个凭证；如请求头存在粘性键则优先命中；若凭证处于冷却期则跳过。
// 选中后会执行到期前预刷新，并根据粘性键回写映射。
func (s *Strategy) Pick(ctx context.Context, hdr http.Header) *credential.Credential {
	if s.credMgr == nil {
		return nil
	}
	// 1) 粘性命中
	if key, src := stickyKeyAndSourceFromHeaders(hdr); key != "" {
		if id, ok := s.getSticky(key); ok {
			if cred, exists := s.credMgr.GetCredentialByID(id); exists && !s.isCooledDown(id) {
				if src == "" {
					src = "auto"
				}
				mon.RoutingStickyHitsTotal.WithLabelValues(src).Inc()
				s.recordPick(PickLog{Time: time.Now(), CredID: cred.ID, Reason: "sticky", StickySource: src})
				return s.PrepareCredential(ctx, cred)
			}
		}
	}
	// 2) 权重选择（简单 P2C）：从全部可用中随机挑两个，按 score 取较优
	creds := s.credMgr.GetAllCredentials()
	candidates := make([]*credential.Credential, 0, len(creds))
	for _, c := range creds {
		if c == nil || c.ID == "" {
			continue
		}
		if s.isCooledDown(c.ID) {
			continue
		}
		if !s.credMgr.HasCapacity(c.ID) {
			continue
		}
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return nil
	}
	var picked *credential.Credential
	var aID, bID string
	var aScore, bScore float64
	if len(candidates) == 1 {
		picked = candidates[0]
	} else {
		i1 := time.Now().UnixNano() % int64(len(candidates))
		i2 := (i1 + 1) % int64(len(candidates))
		a := candidates[i1]
		b := candidates[i2]
		aID, bID = a.ID, b.ID
		aScore, bScore = s.score(a), s.score(b)
		if bScore > aScore {
			picked = b
		} else {
			picked = a
		}
	}
	if picked == nil {
		return nil
	}
	picked = s.PrepareCredential(ctx, picked)
	// 3) 回写粘性
	if key, _ := stickyKeyAndSourceFromHeaders(hdr); key != "" {
		ttl := time.Duration(s.cfg.StickyTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}
		s.setSticky(key, picked.ID, ttl)
	}
	s.recordPick(PickLog{Time: time.Now(), CredID: picked.ID, Reason: "weighted", SampleA: aID, SampleB: bID, ScoreA: aScore, ScoreB: bScore})
	return picked
}

// PickWithInfo 与 Pick 类似，但返回选路日志信息，便于调试/对外暴露。
func (s *Strategy) PickWithInfo(ctx context.Context, hdr http.Header) (*credential.Credential, *PickLog) {
	cred := s.Pick(ctx, hdr)
	if cred == nil {
		return nil, nil
	}
	pl := &PickLog{Time: time.Now(), CredID: cred.ID, Reason: "weighted"}
	recent := s.Picks(10)
	for i := len(recent) - 1; i >= 0; i-- {
		if recent[i].CredID == cred.ID {
			tmp := recent[i]
			pl = &tmp
			break
		}
	}
	return cred, pl
}

func (s *Strategy) score(c *credential.Credential) float64 {
	if c == nil || c.ID == "" {
		return 0
	}
	if s.isCooledDown(c.ID) {
		return 0
	}
	sc := c.GetScore()
	if sc < 0 {
		sc = 0
	}
	if c.DailyLimit > 0 && c.DailyUsage > 0 {
		ratio := float64(c.DailyUsage) / float64(c.DailyLimit)
		if ratio > 0.9 {
			sc *= 0.2
		} else if ratio > 0.75 {
			sc *= 0.6
		}
	}
	return sc
}
