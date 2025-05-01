import React, { useState, useEffect } from 'react';

function Dashboard() {
    const [containers, setContainers] = useState([]);
    const [selectedFile, setSelectedFile] = useState(null);

    useEffect(() => {
        // Fetch containers from the backend
        fetch('/api/containers')
            .then((response) => response.json())
            .then((data) => setContainers(data))
            .catch((error) => console.error('Error fetching containers:', error));
    }, []);

    const handleFileChange = (event) => {
        setSelectedFile(event.target.files[0]);
    };

    const handleFileUpload = (event) => {
        event.preventDefault();
        const formData = new FormData();
        formData.append('yamlFile', selectedFile);

        fetch('/upload', {
            method: 'POST',
            body: formData,
        })
            .then((response) => response.json())
            .then((data) => {
                alert('File uploaded successfully');
                window.location.reload();
            })
            .catch((error) => alert('Error uploading file: ' + error));
    };

    const handleDeleteContainer = (id) => {
        if (window.confirm('Are you sure you want to delete this container?')) {
            fetch(`/api/containers/${id}`, { method: 'DELETE' })
                .then(() => {
                    alert('Container deleted successfully');
                    setContainers((prev) => prev.filter((container) => container.ID !== id));
                })
                .catch((error) => alert('Error deleting container: ' + error));
        }
    };

    return (
        <div className="container">
            <header>
                <h1>DockFormer Dashboard</h1>
            </header>

            <section className="upload-section">
                <h2>Upload YAML Configuration</h2>
                <form onSubmit={handleFileUpload}>
                    <div className="file-input">
                        <input type="file" onChange={handleFileChange} accept=".yaml,.yml" />
                        <label>{selectedFile ? selectedFile.name : 'Select YAML File'}</label>
                    </div>
                    <button type="submit" className="btn btn-primary">Upload</button>
                </form>
            </section>

            <section className="container-list">
                <h2>Containers</h2>
                <table>
                    <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Image</th>
                        <th>Status</th>
                        <th>Ports</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                    </thead>
                    <tbody>
                    {containers.length > 0 ? (
                        containers.map((container) => (
                            <tr key={container.ID} className={`status-${container.Status}`}>
                                <td>{container.ID}</td>
                                <td>{container.Name}</td>
                                <td>{container.Image}</td>
                                <td><span className="status-badge">{container.Status}</span></td>
                                <td>{container.Ports}</td>
                                <td>{new Date(container.CreatedAt).toLocaleString()}</td>
                                <td className="actions">
                                    {container.Status === 'running' ? (
                                        <button className="btn btn-sm btn-warning">Stop</button>
                                    ) : (
                                         <button className="btn btn-sm btn-success">Start</button>
                                     )}
                                    <button className="btn btn-sm btn-info">Restart</button>
                                    <a href={`/logs/${container.ID}`} className="btn btn-sm btn-secondary">Logs</a>
                                    <button
                                        className="btn btn-sm btn-danger"
                                        onClick={() => handleDeleteContainer(container.ID)}
                                    >
                                        Delete
                                    </button>
                                </td>
                            </tr>
                        ))
                    ) : (
                         <tr>
                             <td colSpan="7" className="empty-message">No containers found</td>
                         </tr>
                     )}
                    </tbody>
                </table>
            </section>
        </div>
    );
}

export default Dashboard;