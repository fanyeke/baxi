import { Routes, Route, Navigate } from "react-router-dom"
import Layout from "./components/Layout"
import Dashboard from "./pages/Dashboard"
import Alerts from "./pages/Alerts"
import Tasks from "./pages/Tasks"
import Outbox from "./pages/Outbox"
import Logs from "./pages/Logs"
import Feishu from "./pages/Feishu"
import Pipeline from "./pages/Pipeline"
import Governance from "./pages/Governance"
import AgentLogs from "./pages/AgentLogs"
import CaseDetail from "./pages/CaseDetail"
import AuditTimeline from "./pages/AuditTimeline"
import PolicyInspector from "./pages/PolicyInspector"

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/tasks" element={<Tasks />} />
        <Route path="/outbox" element={<Outbox />} />
        <Route path="/logs" element={<Logs />} />
        <Route path="/feishu" element={<Feishu />} />
        <Route path="/pipeline" element={<Pipeline />} />
        <Route path="/governance" element={<Governance />} />
        <Route path="/agent-logs" element={<AgentLogs />} />
        <Route path="/cases/:id" element={<CaseDetail />} />
        <Route path="/audit-timeline" element={<AuditTimeline />} />
        <Route path="/policy-inspector" element={<PolicyInspector />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default App
