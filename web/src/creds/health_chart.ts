/**
 * 健康评分可视化模块
 * 提供健康评分的图表展示功能
 */

import type { Credential } from './types.js';
import { getHealthLevel, getHealthColor } from './health.js';

export interface HealthChartOptions {
    width?: number;
    height?: number;
    showLegend?: boolean;
    showLabels?: boolean;
}

/**
 * 渲染健康评分分布饼图
 */
export function renderHealthDistributionChart(
    credentials: Credential[],
    options: HealthChartOptions = {}
): string {
    const {
        width = 300,
        height = 300,
        showLegend = true,
    } = options;

    // 统计各健康等级的数量
    const distribution = {
        excellent: 0,
        good: 0,
        fair: 0,
        poor: 0,
        unknown: 0,
    };

    credentials.forEach(cred => {
        const score = Number(cred.health_score) || 0;
        const level = getHealthLevel(score);
        distribution[level as keyof typeof distribution]++;
    });

    const total = credentials.length;
    if (total === 0) {
        return '<div class="health-chart-empty">暂无数据</div>';
    }

    // 计算百分比
    const percentages = {
        excellent: Math.round((distribution.excellent / total) * 100),
        good: Math.round((distribution.good / total) * 100),
        fair: Math.round((distribution.fair / total) * 100),
        poor: Math.round((distribution.poor / total) * 100),
        unknown: Math.round((distribution.unknown / total) * 100),
    };

    const colors = {
        excellent: '#10b981',
        good: '#3b82f6',
        fair: '#f59e0b',
        poor: '#ef4444',
        unknown: '#6b7280',
    };

    const labels = {
        excellent: '优秀',
        good: '良好',
        fair: '一般',
        poor: '较差',
        unknown: '未知',
    };

    return `
        <div class="health-chart">
            <div class="health-chart-title">健康评分分布</div>
            <div class="health-chart-body">
                <svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
                    ${renderPieChart(distribution, colors, width, height)}
                </svg>
                ${showLegend ? renderLegend(distribution, percentages, colors, labels) : ''}
            </div>
        </div>
    `;
}

/**
 * 渲染饼图 SVG
 */
function renderPieChart(
    distribution: Record<string, number>,
    colors: Record<string, string>,
    width: number,
    height: number
): string {
    const total = Object.values(distribution).reduce((sum, val) => sum + val, 0);
    if (total === 0) return '';

    const cx = width / 2;
    const cy = height / 2;
    const radius = Math.min(width, height) / 2 - 10;

    let currentAngle = -90; // 从顶部开始
    const slices: string[] = [];

    Object.entries(distribution).forEach(([level, count]) => {
        if (count === 0) return;

        const percentage = count / total;
        const angle = percentage * 360;
        const endAngle = currentAngle + angle;

        const startRad = (currentAngle * Math.PI) / 180;
        const endRad = (endAngle * Math.PI) / 180;

        const x1 = cx + radius * Math.cos(startRad);
        const y1 = cy + radius * Math.sin(startRad);
        const x2 = cx + radius * Math.cos(endRad);
        const y2 = cy + radius * Math.sin(endRad);

        const largeArc = angle > 180 ? 1 : 0;

        const pathData = [
            `M ${cx} ${cy}`,
            `L ${x1} ${y1}`,
            `A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2}`,
            'Z',
        ].join(' ');

        slices.push(`
            <path
                d="${pathData}"
                fill="${colors[level]}"
                stroke="white"
                stroke-width="2"
            >
                <title>${level}: ${count} (${Math.round(percentage * 100)}%)</title>
            </path>
        `);

        currentAngle = endAngle;
    });

    return slices.join('');
}

/**
 * 渲染图例
 */
function renderLegend(
    distribution: Record<string, number>,
    percentages: Record<string, number>,
    colors: Record<string, string>,
    labels: Record<string, string>
): string {
    const items = Object.entries(distribution)
        .filter(([_, count]) => count > 0)
        .map(([level, count]) => `
            <div class="health-legend-item">
                <span class="health-legend-color" style="background-color: ${colors[level]}"></span>
                <span class="health-legend-label">${labels[level]}</span>
                <span class="health-legend-value">${count} (${percentages[level]}%)</span>
            </div>
        `);

    return `
        <div class="health-legend">
            ${items.join('')}
        </div>
    `;
}

/**
 * 渲染健康评分趋势图（简化版柱状图）
 */
export function renderHealthTrendChart(
    credentials: Credential[],
    options: HealthChartOptions = {}
): string {
    const {
        width = 400,
        height = 200,
    } = options;

    // 按健康评分分组（0-20, 20-40, 40-60, 60-80, 80-100）
    const buckets = [0, 0, 0, 0, 0];
    const bucketLabels = ['0-20%', '20-40%', '40-60%', '60-80%', '80-100%'];

    credentials.forEach(cred => {
        const score = Number(cred.health_score) || 0;
        const bucketIndex = Math.min(Math.floor(score * 5), 4);
        buckets[bucketIndex]++;
    });

    const maxCount = Math.max(...buckets, 1);

    return `
        <div class="health-chart">
            <div class="health-chart-title">健康评分分布</div>
            <div class="health-chart-body">
                <svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
                    ${renderBarChart(buckets, bucketLabels, maxCount, width, height)}
                </svg>
            </div>
        </div>
    `;
}

/**
 * 渲染柱状图 SVG
 */
function renderBarChart(
    buckets: number[],
    labels: string[],
    maxCount: number,
    width: number,
    height: number
): string {
    const padding = 40;
    const chartWidth = width - padding * 2;
    const chartHeight = height - padding * 2;
    const barWidth = chartWidth / buckets.length;

    const bars = buckets.map((count, index) => {
        const barHeight = (count / maxCount) * chartHeight;
        const x = padding + index * barWidth;
        const y = padding + chartHeight - barHeight;

        const color = getBarColor(index);

        return `
            <g>
                <rect
                    x="${x + 5}"
                    y="${y}"
                    width="${barWidth - 10}"
                    height="${barHeight}"
                    fill="${color}"
                    rx="4"
                >
                    <title>${labels[index]}: ${count}</title>
                </rect>
                <text
                    x="${x + barWidth / 2}"
                    y="${y - 5}"
                    text-anchor="middle"
                    font-size="12"
                    fill="#666"
                >
                    ${count}
                </text>
                <text
                    x="${x + barWidth / 2}"
                    y="${padding + chartHeight + 20}"
                    text-anchor="middle"
                    font-size="10"
                    fill="#999"
                >
                    ${labels[index]}
                </text>
            </g>
        `;
    });

    return bars.join('');
}

/**
 * 获取柱状图颜色
 */
function getBarColor(index: number): string {
    const colors = ['#ef4444', '#f59e0b', '#3b82f6', '#10b981', '#059669'];
    return colors[index] || '#6b7280';
}

/**
 * 渲染健康评分仪表盘
 */
export function renderHealthGauge(score: number, options: HealthChartOptions = {}): string {
    const {
        width = 200,
        height = 150,
    } = options;

    const level = getHealthLevel(score);
    const color = getHealthColor(level);
    const percentage = Math.round(score * 100);

    const cx = width / 2;
    const cy = height - 20;
    const radius = Math.min(width, height) / 2 - 20;

    // 计算指针角度（-90度到90度）
    const angle = -90 + (score * 180);
    const angleRad = (angle * Math.PI) / 180;
    const needleX = cx + radius * 0.8 * Math.cos(angleRad);
    const needleY = cy + radius * 0.8 * Math.sin(angleRad);

    return `
        <div class="health-gauge">
            <svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
                <!-- 背景弧 -->
                <path
                    d="M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${cx + radius} ${cy}"
                    fill="none"
                    stroke="#e5e7eb"
                    stroke-width="20"
                    stroke-linecap="round"
                />
                <!-- 彩色弧 -->
                <path
                    d="M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${needleX} ${needleY}"
                    fill="none"
                    stroke="${color}"
                    stroke-width="20"
                    stroke-linecap="round"
                />
                <!-- 指针 -->
                <line
                    x1="${cx}"
                    y1="${cy}"
                    x2="${needleX}"
                    y2="${needleY}"
                    stroke="#333"
                    stroke-width="3"
                    stroke-linecap="round"
                />
                <circle cx="${cx}" cy="${cy}" r="6" fill="#333" />
                <!-- 分数文本 -->
                <text
                    x="${cx}"
                    y="${cy - radius - 10}"
                    text-anchor="middle"
                    font-size="24"
                    font-weight="bold"
                    fill="${color}"
                >
                    ${percentage}%
                </text>
                <text
                    x="${cx}"
                    y="${cy - radius + 10}"
                    text-anchor="middle"
                    font-size="14"
                    fill="#666"
                >
                    ${level}
                </text>
            </svg>
        </div>
    `;
}

