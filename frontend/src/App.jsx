import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Logs from './pages/Logs';
import ErrorPage from './pages/ErrorPage';

function App() {
    return (
        <Router>
            <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/logs/:id" element={<Logs />} />
                <Route path="/error" element={<ErrorPage />} />
            </Routes>
        </Router>
    );
}

export default App;