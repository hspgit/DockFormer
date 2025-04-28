// View container details
function viewContainer(id) {
    window.location.href = `/api/containers/${id}`;
}

// Delete container
function deleteContainer(id) {
    if (confirm('Are you sure you want to delete this container?')) {
        fetch(`/api/containers/${id}`, {
            method: 'DELETE',
        })
        .then(response => response.json())
        .then(data => {
            alert('Container deleted successfully');
            window.location.reload();
        })
        .catch(error => {
            alert('Error deleting container: ' + error);
        });
    }
}

// Add file name to label when file is selected
document.addEventListener('DOMContentLoaded', function() {
    const fileInput = document.getElementById('yamlFile');
    const fileLabel = document.querySelector('label[for="yamlFile"]');

    if (fileInput && fileLabel) {
        fileInput.addEventListener('change', function() {
            if (this.files.length > 0) {
                fileLabel.textContent = this.files[0].name;
            } else {
                fileLabel.textContent = 'Select YAML File';
            }
        });
    }
});