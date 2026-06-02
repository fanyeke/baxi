import { Routes, Route, Navigate } from "react-router-dom"
import Layout from "./components/Layout"
import { ErrorBoundary } from "./components/ErrorBoundary"
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
import DecisionReview from "./pages/DecisionReview"
import SandboxCompare from "./pages/SandboxCompare"

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<ErrorBoundary><Dashboard /></ErrorBoundary>} />
        <Route path="/alerts" element={<ErrorBoundary><Alerts /></ErrorBoundary>} />
        <Route path="/tasks" element={<ErrorBoundary><Tasks /></ErrorBoundary>} />
        <Route path="/outbox" element={<ErrorBoundary><Outbox /></ErrorBoundary>} />
        <Route path="/logs" element={<ErrorBoundary><Logs /></ErrorBoundary>} />
        <Route path="/feishu" element={<ErrorBoundary><Feishu /></ErrorBoundary>} />
        <Route path="/pipeline" element={<ErrorBoundary><Pipeline /></ErrorBoundary>} />
        <Route path="/governance" element={<ErrorBoundary><Governance /></ErrorBoundary>} />
        <Route path="/agent-logs" element={<ErrorBoundary><AgentLogs /></ErrorBoundary>} />
        <Route path="/cases/:id" element={<ErrorBoundary><CaseDetail /></ErrorBoundary>} />
        <Route path="/audit-timeline" element={<ErrorBoundary><AuditTimeline /></ErrorBoundary>} />
        <Route path="/policy-inspector" element={<ErrorBoundary><PolicyInspector /></ErrorBoundary>} />
        <Route path="/decision-review" element={<ErrorBoundary><DecisionReview /></ErrorBoundary>} />
        <Route path="/sandbox" element={<ErrorBoundary><SandboxCompare /></ErrorBoundary>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default App
