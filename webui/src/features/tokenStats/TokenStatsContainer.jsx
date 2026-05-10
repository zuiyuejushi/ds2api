import { useEffect, useMemo, useRef, useState } from 'react'
import { Activity, DollarSign, Loader2, RefreshCw, TrendingUp, Zap } from 'lucide-react'
import clsx from 'clsx'

import { useI18n } from '../../i18n'

const RANGE_OPTIONS = [
    { key: '30s', label: '30s' },
    { key: '24h', label: '24小时' },
    { key: '7d', label: '7天' },
    { key: '30d', label: '30天' },
]

const MODEL_FILTERS = [
    { key: 'all', label: '全部' },
    { key: 'deepseek-v4-flash', label: 'Flash' },
    { key: 'deepseek-v4-pro', label: 'Pro' },
]

function formatNumber(num) {
    if (num >= 1_000_000_000) {
        return (num / 1_000_000_000).toFixed(2) + 'B'
    }
    if (num >= 1_000_000) {
        return (num / 1_000_000).toFixed(2) + 'M'
    }
    if (num >= 1_000) {
        return (num / 1_000).toFixed(1) + 'k'
    }
    return num.toString()
}

function formatCurrency(num) {
    return '$' + num.toFixed(4)
}

function formatDateTime(timestamp, range) {
    const date = new Date(timestamp)
    if (range === '30s') {
        return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    }
    if (range === '24h') {
        return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    }
    return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

// Simple SVG Line Chart Component
function LineChart({ data, range }) {
    const containerRef = useRef(null)
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 })

    useEffect(() => {
        if (!containerRef.current) return
        const resizeObserver = new ResizeObserver((entries) => {
            for (const entry of entries) {
                setDimensions({
                    width: entry.contentRect.width,
                    height: entry.contentRect.height,
                })
            }
        })
        resizeObserver.observe(containerRef.current)
        return () => resizeObserver.disconnect()
    }, [])

    const { path, areaPath, maxValue, yTicks } = useMemo(() => {
        if (!data.length || dimensions.width === 0 || dimensions.height === 0) {
            return { path: '', areaPath: '', maxValue: 0, yTicks: [] }
        }

        const padding = { top: 20, right: 60, bottom: 40, left: 10 }
        const chartWidth = dimensions.width - padding.left - padding.right
        const chartHeight = dimensions.height - padding.top - padding.bottom

        const maxTokens = Math.max(...data.map((d) => d.total_tokens), 1)
        const maxCost = Math.max(...data.map((d) => d.cost), 0.01)
        const maxVal = Math.max(maxTokens, maxCost * 1000000)

        const yTicks = [0, maxVal * 0.25, maxVal * 0.5, maxVal * 0.75, maxVal]

        const getX = (i) => padding.left + (i / (data.length - 1)) * chartWidth
        const getY = (val) => padding.top + chartHeight - (val / maxVal) * chartHeight

        // Total tokens line (solid)
        let linePath = ''
        data.forEach((point, i) => {
            const x = getX(i)
            const y = getY(point.total_tokens)
            if (i === 0) {
                linePath += `M ${x} ${y}`
            } else {
                // Simple bezier curve
                const prevX = getX(i - 1)
                const prevY = getY(data[i - 1].total_tokens)
                const cpX1 = prevX + (x - prevX) / 3
                const cpX2 = prevX + (2 * (x - prevX)) / 3
                linePath += ` C ${cpX1} ${prevY}, ${cpX2} ${y}, ${x} ${y}`
            }
        })

        // Area under the line
        const areaPath =
            linePath +
            ` L ${getX(data.length - 1)} ${padding.top + chartHeight}` +
            ` L ${padding.left} ${padding.top + chartHeight} Z`

        return { path: linePath, areaPath, maxValue: maxVal, yTicks }
    }, [data, dimensions])

    if (!data.length) {
        return (
            <div className="h-full flex items-center justify-center text-muted-foreground">
                暂无数据
            </div>
        )
    }

    return (
        <div ref={containerRef} className="relative w-full h-full">
            <svg className="absolute inset-0 w-full h-full">
                <defs>
                    <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="rgb(59, 130, 246)" stopOpacity="0.3" />
                        <stop offset="100%" stopColor="rgb(59, 130, 246)" stopOpacity="0.05" />
                    </linearGradient>
                </defs>

                {/* Y-axis grid lines */}
                {yTicks.map((tick, i) => (
                    <g key={i}>
                        <line
                            x1="10"
                            y1={20 + (dimensions.height - 60) * (1 - i / 4)}
                            x2={dimensions.width - 60}
                            y2={20 + (dimensions.height - 60) * (1 - i / 4)}
                            stroke="rgba(148, 163, 184, 0.2)"
                            strokeDasharray="4,4"
                        />
                        <text
                            x={dimensions.width - 50}
                            y={20 + (dimensions.height - 60) * (1 - i / 4) + 4}
                            fill="rgba(148, 163, 184, 0.6)"
                            fontSize="10"
                        >
                            {formatNumber(tick)}
                        </text>
                    </g>
                ))}

                {/* Area */}
                {areaPath && <path d={areaPath} fill="url(#areaGradient)" />}

                {/* Line */}
                {path && (
                    <path
                        d={path}
                        fill="none"
                        stroke="rgb(59, 130, 246)"
                        strokeWidth="2"
                        strokeLinecap="round"
                    />
                )}

                {/* Data points */}
                {data.map((point, i) => {
                    const x = 10 + (i / (data.length - 1)) * (dimensions.width - 70)
                    const y = 20 + (dimensions.height - 60) * (1 - point.total_tokens / (maxValue || 1))
                    return (
                        <circle
                            key={i}
                            cx={x}
                            cy={y}
                            r="3"
                            fill="rgb(59, 130, 246)"
                            stroke="white"
                            strokeWidth="1"
                        />
                    )
                })}
            </svg>

            {/* X-axis labels */}
            <div className="absolute bottom-0 left-10 right-[60px] flex justify-between text-[10px] text-muted-foreground">
                {data.length > 0 && (
                    <>
                        <span>{formatDateTime(data[0].timestamp, range)}</span>
                        <span>{formatDateTime(data[Math.floor(data.length / 2)].timestamp, range)}</span>
                        <span>{formatDateTime(data[data.length - 1].timestamp, range)}</span>
                    </>
                )}
            </div>
        </div>
    )
}

export default function TokenStatsContainer({ authFetch, onMessage }) {
    const { t } = useI18n()
    const apiFetch = authFetch || fetch

    const [loading, setLoading] = useState(true)
    const [refreshing, setRefreshing] = useState(false)
    const [range, setRange] = useState('30d')
    const [modelFilter, setModelFilter] = useState('all')
    const [stats, setStats] = useState(null)

    const loadStats = async ({ silent = false } = {}) => {
        if (!silent) setLoading(true)
        try {
            const res = await apiFetch(`/admin/token-stats?range=${range}`)
            if (!res.ok) {
                throw new Error('加载统计失败')
            }
            const data = await res.json()
            setStats(data)
        } catch (error) {
            onMessage?.('error', error.message)
        } finally {
            setLoading(false)
            setRefreshing(false)
        }
    }

    useEffect(() => {
        loadStats()
    }, [range])

    const handleRefresh = () => {
        setRefreshing(true)
        loadStats({ silent: true }).finally(() => setRefreshing(false))
    }

    const filteredPoints = useMemo(() => {
        if (!stats?.points) return []
        return stats.points
    }, [stats])

    if (loading) {
        return (
            <div className="h-[calc(100vh-200px)] rounded-2xl border border-border bg-card shadow-sm flex items-center justify-center">
                <div className="flex items-center gap-3 text-sm text-muted-foreground">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    加载中...
                </div>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="rounded-2xl border border-border bg-card shadow-sm p-4 lg:p-5">
                <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
                    <div>
                        <div className="text-lg font-semibold text-foreground">使用统计</div>
                        <div className="text-xs text-muted-foreground mt-1">查看 AI 模型的使用情况和成本统计</div>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                        {/* Range selector */}
                        <div className="inline-flex items-center rounded-xl border border-border bg-background p-1">
                            {RANGE_OPTIONS.map((opt) => (
                                <button
                                    key={opt.key}
                                    onClick={() => setRange(opt.key)}
                                    className={clsx(
                                        'h-8 px-3 rounded-lg text-sm font-medium transition-colors',
                                        range === opt.key
                                            ? 'bg-primary text-primary-foreground'
                                            : 'text-muted-foreground hover:text-foreground hover:bg-secondary/60'
                                    )}
                                >
                                    {opt.label}
                                </button>
                            ))}
                        </div>
                        <button
                            onClick={handleRefresh}
                            disabled={refreshing}
                            className="h-9 w-9 rounded-lg border border-border bg-background text-muted-foreground hover:text-foreground hover:bg-secondary/70 flex items-center justify-center"
                        >
                            {refreshing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
                        </button>
                    </div>
                </div>

                {/* Model filters */}
                <div className="flex items-center gap-2 mt-4">
                    {MODEL_FILTERS.map((filter) => (
                        <button
                            key={filter.key}
                            onClick={() => setModelFilter(filter.key)}
                            className={clsx(
                                'h-8 px-4 rounded-full text-sm font-medium transition-colors border',
                                modelFilter === filter.key
                                    ? 'bg-primary/10 border-primary text-primary'
                                    : 'bg-background border-border text-muted-foreground hover:text-foreground hover:bg-secondary/60'
                            )}
                        >
                            {filter.label}
                        </button>
                    ))}
                </div>

                {/* Summary line */}
                <div className="flex items-center gap-4 mt-4 text-sm text-muted-foreground">
                    <span>{stats?.total_requests || 0} 次请求</span>
                    <span className="text-border">|</span>
                    <span>{formatCurrency(stats?.total_cost || 0)} 总成本</span>
                </div>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
                {/* Total Requests */}
                <div className="rounded-2xl border border-border bg-card shadow-sm p-5">
                    <div className="flex items-center justify-between">
                        <div className="text-sm text-muted-foreground">总请求数</div>
                        <div className="w-8 h-8 rounded-lg bg-blue-500/10 flex items-center justify-center">
                            <Activity className="w-4 h-4 text-blue-500" />
                        </div>
                    </div>
                    <div className="text-2xl font-bold text-foreground mt-3">
                        {formatNumber(stats?.total_requests || 0)}
                    </div>
                </div>

                {/* Total Cost */}
                <div className="rounded-2xl border border-border bg-card shadow-sm p-5">
                    <div className="flex items-center justify-between">
                        <div className="text-sm text-muted-foreground">总成本</div>
                        <div className="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center">
                            <DollarSign className="w-4 h-4 text-emerald-500" />
                        </div>
                    </div>
                    <div className="text-2xl font-bold text-foreground mt-3">
                        {formatCurrency(stats?.total_cost || 0)}
                    </div>
                </div>

                {/* Total Tokens */}
                <div className="rounded-2xl border border-border bg-card shadow-sm p-5">
                    <div className="flex items-center justify-between">
                        <div className="text-sm text-muted-foreground">总 Token 数</div>
                        <div className="w-8 h-8 rounded-lg bg-purple-500/10 flex items-center justify-center">
                            <Zap className="w-4 h-4 text-purple-500" />
                        </div>
                    </div>
                    <div className="text-2xl font-bold text-foreground mt-3">
                        {formatNumber(stats?.total_tokens || 0)}
                    </div>
                    <div className="mt-2 space-y-1">
                        <div className="flex justify-between text-xs text-muted-foreground">
                            <span>Input</span>
                            <span>{formatNumber(stats?.total_prompt_tokens || 0)}</span>
                        </div>
                        <div className="flex justify-between text-xs text-muted-foreground">
                            <span>Output</span>
                            <span>{formatNumber(stats?.total_output_tokens || 0)}</span>
                        </div>
                    </div>
                </div>

                {/* Cached Tokens */}
                <div className="rounded-2xl border border-border bg-card shadow-sm p-5">
                    <div className="flex items-center justify-between">
                        <div className="text-sm text-muted-foreground">缓存 Token</div>
                        <div className="w-8 h-8 rounded-lg bg-amber-500/10 flex items-center justify-center">
                            <TrendingUp className="w-4 h-4 text-amber-500" />
                        </div>
                    </div>
                    <div className="text-2xl font-bold text-foreground mt-3">
                        {formatNumber(stats?.cached_tokens || 0)}
                    </div>
                    <div className="mt-2 space-y-1">
                        <div className="flex justify-between text-xs text-muted-foreground">
                            <span>创建</span>
                            <span>0</span>
                        </div>
                        <div className="flex justify-between text-xs text-muted-foreground">
                            <span>命中</span>
                            <span>{formatNumber(stats?.cached_tokens || 0)}</span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Chart */}
            <div className="rounded-2xl border border-border bg-card shadow-sm p-5">
                <div className="flex items-center justify-between mb-4">
                    <div className="text-sm font-semibold text-foreground">使用趋势</div>
                    <div className="text-xs text-muted-foreground">
                        {range === '30s' && '过去 30 秒'}
                        {range === '24h' && '过去 24 小时'}
                        {range === '7d' && '过去 7 天'}
                        {range === '30d' && '过去 30 天'}
                    </div>
                </div>
                <div className="h-[300px]">
                    <LineChart data={filteredPoints} range={range} />
                </div>
            </div>
        </div>
    )
}
