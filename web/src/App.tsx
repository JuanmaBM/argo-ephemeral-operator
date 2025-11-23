import { Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './components/Layout/AppLayout';
import { Dashboard } from './pages/Dashboard/Dashboard';
import { EnvironmentDetail } from './pages/EnvironmentDetail/EnvironmentDetail';
import { Settings } from './pages/Settings/Settings';

function App() {
  return (
    <AppLayout>
      <Routes>
        <Route path="/" element={<Navigate to="/environments" replace />} />
        <Route path="/environments" element={<Dashboard />} />
        <Route path="/environments/:name" element={<EnvironmentDetail />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </AppLayout>
  );
}

export default App;

