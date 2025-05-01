# DockFormer

DockFormer is a minimal application designed to spin up Docker containers based on provided configuration files. It features a Go-based backend and a React-based frontend for managing and monitoring containers.

## Features

- Upload YAML configuration files to define Docker containers.
- View a list of running containers with details such as name, image, status, and ports.
- Perform actions on containers: start, stop, restart, delete, and view logs.
- View detailed logs for individual containers.

## Project Structure
/
```plaintext
├── backend/
│   ├── go.mod
│   ├── internal/
│   │   ├── database/ # Database connection and migrations
│   │   └── server/   # HTTP server and API routes
│   ├── main.go     # Entry point for the backend
│   └── web/        # Static files and templates for production
├── frontend/
│   ├── public/     # Static assets
│   ├── src/        # React source code
│   │   ├── components/ # Reusable components
│   │   └── pages/      # Page-level components
│   ├── package.json  # NPM dependencies and scripts
│   └── vite.config.js  # Vite configuration
└── README.md       # This file
```


## Prerequisites

- [Go](https://golang.org/) (1.20+ recommended)
- [Node.js](https://nodejs.org/) (16+ recommended)
- Docker (for container management)
- PostgreSQL (for database)

## Setup Instructions

### 1. Backend Setup

1.  Clone the repository:

    ```bash
    git clone [https://github.com/hspgit/DockFormer.git](https://github.com/hspgit/DockFormer.git)
    cd DockFormer/backend
    ```

2.  Create a `.env` file in the `backend` directory with the following content:

    ```env
    DATABASE_URL=postgres://user:password@localhost:5432/db_name?sslmode=disable
    ```

    Replace `user`, `password`, `localhost:5432`, and `db_name` with your PostgreSQL database credentials and connection details.

3.  Run the backend:

    ```bash
    go run main.go
    ```

    The backend server should start, typically listening on `http://localhost:8080`.

### 2. Frontend Setup

1.  Navigate to the frontend directory:

    ```bash
    cd ../frontend
    ```

2.  Install dependencies:

    ```bash
    npm install
    ```

3.  Start the development server:

    ```bash
    npm run dev
    ```

    The frontend development server should start, typically listening on `http://localhost:5173`. The frontend is configured to proxy API requests to the backend (see `vite.config.js`).

### 3. Running Both Services

Ensure both the backend (`go run main.go` in the `/backend` directory) and the frontend (`npm run dev` in the `/frontend` directory) are running simultaneously for the full application to function.

-   Backend default address: `http://localhost:8080`
-   Frontend default address: `http://localhost:5173`

### 4. Build for Production

To build the frontend for production and serve it with the Go backend:

1.  Build the React app from the `frontend` directory:

    ```bash
    npm run build
    ```

    This will create a `dist` directory inside `frontend`.

2.  Configure your Go backend to serve the static files from the `frontend/dist` directory. An example using `gorilla/mux` might look like this:

    ```go
    // In your backend server setup (e.g., main.go or server package)
    r := mux.NewRouter()
    // ... your API routes ...

    // Serve the static frontend files
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/dist")))

    // Start the server
    // log.Fatal(http.ListenAndServe(":8080", r))
    ```
    *(Note: This is a conceptual example. The exact implementation depends on your backend server framework.)*

## API Endpoints

-   `GET /api/containers`: Fetch a list of running containers.
-   `POST /upload`: Upload a YAML configuration file to create containers.
-   `DELETE /api/containers/:id`: Delete a specific container by ID.
-   `GET /api/containers/:id/logs`: Fetch logs for a specific container by ID.
-   `POST /api/containers/:id/start`: Start a specific container by ID.
-   `POST /api/containers/:id/stop`: Stop a specific container by ID.
-   `POST /api/containers/:id/restart`: Restart a specific container by ID.

## Technologies Used

-   **Backend:** Go, GORM (for database interaction), PostgreSQL
-   **Frontend:** React, Vite
-   **Containerization:** Docker
-   **Styling:** HTML/CSS

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.