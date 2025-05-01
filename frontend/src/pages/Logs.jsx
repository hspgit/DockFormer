import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';

function Logs() {
  const { id } = useParams();
  const [logs, setLogs] = useState('');
  const [containerInfo, setContainerInfo] = useState(null);

  useEffect(() => {
    // Fetch container logs
    fetch(`/api/containers/${id}/logs`)
      .then((response) => response.json())
      .then((data) => {
        setLogs(data.logs);
        setContainerInfo(data.container);
      })
      .catch((error) => console.error('Error fetching logs:', error));
  }, [id]);

  return (
    <div className="container">
      <header>
        <h1>Container Logs</h1>
      </header>

      {containerInfo && (
        <div className="container-info">
          <h2>{containerInfo.Name}</h2>
          <p><strong>Status:</strong> {containerInfo.Status}</p>
          <p><strong>Image:</strong> {containerInfo.Image}</p>
          <p><strong>Created:</strong> {new Date(containerInfo.CreatedAt).toLocaleString()}</p>
          <a href="/" className="btn">Back to Dashboard</a>
        </div>
      )}

      <div className="logs-container">
        <pre>{logs}</pre>
      </div>
    </div>
  );
}

export default Logs;