import { Outlet, NavLink } from "react-router-dom"
import { useState } from "react"

const NAV_ITEMS = [
  { to: "/", label: "总览", icon: "◈" },
  { to: "/alerts", label: "告警中心", icon: "⚠" },
  { to: "/tasks", label: "任务中心", icon: "☰" },
  { to: "/outbox", label: "Outbox 分发", icon: "⇶" },
  { to: "/logs", label: "日志诊断", icon: "📋" },
  { to: "/feishu", label: "飞书同步", icon: "🔗" },
  { to: "/pipeline", label: "运行管道", icon: "▶" },
  { to: "/governance", label: "治理中心", icon: "🏛" },
  { to: "/agent-logs", label: "Agent 日志", icon: "🤖" },
  { to: "/decision-review", label: "决策审查", icon: "⚖" },
  { to: "/sandbox", label: "沙箱对比", icon: "🧪" },
  { to: "/audit-timeline", label: "审计时间线", icon: "⏱" },
  { to: "/policy-inspector", label: "策略审查", icon: "🔍" },
]

export default function Layout() {
  const [token, setToken] = useState(() => sessionStorage.getItem("API_BEARER_TOKEN") || "")

  const handleTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    setToken(val)
    sessionStorage.setItem("API_BEARER_TOKEN", val)
  }

  return (
    <div className="flex h-screen bg-background">
      <aside className="w-56 border-r shrink-0 flex flex-col bg-sidebar text-sidebar-foreground">
        <div className="p-4 font-bold text-lg border-b">Olist 决策中台</div>
        <nav className="flex-1 p-2 space-y-1">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) =>
                `flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                    : "hover:bg-sidebar-accent/50"
                }`
              }
            >
              <span>{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t">
          <label className="text-xs text-muted-foreground block mb-1">API Token</label>
          <input
            type="password"
            value={token}
            onChange={handleTokenChange}
            placeholder="Bearer Token"
            className="w-full text-xs px-2 py-1 border rounded bg-background"
          />
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
