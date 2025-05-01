import React from 'react';

function ErrorPage() {
  return (
    <div className="container error-page">
      <header>
        <h1>Error</h1>
      </header>
      <div className="error-message">
        <p>An error occurred. Please try again later.</p>
      </div>
      <div className="actions">
        <a href="/" className="btn btn-primary">Back to Dashboard</a>
      </div>
    </div>
  );
}

export default ErrorPage;