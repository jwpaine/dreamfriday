document.addEventListener('click', function (e) {
    var pid = e.target.getAttribute('pid');
    if (pid) {
        console.log(pid);

        if (e.target.tagName.toLowerCase() === 'a') {
            e.preventDefault(); // Prevent default link behavior
        }

        fetch(`/preview/element/${pid}`)
            .then(response => response.json())
            .then(data => {
                console.log(data);
                // Display the edit form overlay with the retrieved JSON data
                displayEditForm(pid, null, data);
            })
            .catch(error => {
                console.error('Error fetching preview element:', error);
            });
    }
});

function displayEditForm(pid, page, data) {
    // Create overlay container
    var overlay = document.createElement('div');
    overlay.style.position = 'fixed';
    overlay.style.top = '0';
    overlay.style.left = '0';
    overlay.style.width = '100%';
    overlay.style.height = '100%';
    overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
    overlay.style.display = 'flex';
    overlay.style.alignItems = 'center';
    overlay.style.justifyContent = 'center';
    overlay.style.zIndex = '1000';

    // Create form container
    var formContainer = document.createElement('div');
    formContainer.style.backgroundColor = 'rgba(255,255,255,0.3)';
    formContainer.style.padding = '20px';
    formContainer.style.borderRadius = '5px';

    // Create textarea and fill with formatted JSON
    var textarea = document.createElement('textarea');
    textarea.style.width = '400px';
    textarea.style.height = '200px';
    textarea.style.background = 'rgba(255,255,255,0.8)';
    textarea.value = JSON.stringify(data, null, 2);

    // Create the update button
    var updateElementButton = document.createElement('button');
    updateElementButton.textContent = 'Update';

    var updatePageButton = document.createElement('button');
    updatePageButton.textContent = 'Update Page';
    // hide initially
    updatePageButton.style.display = 'none';

    var showAllButton = document.createElement('button');
    showAllButton.textContent = 'Show All';

    // Create the cancel button
    var cancelButton = document.createElement('button');
    cancelButton.textContent = 'Cancel';
    cancelButton.style.marginLeft = '10px';

    // Append textarea and buttons to the form container
    formContainer.appendChild(textarea);
    formContainer.appendChild(document.createElement('br'));
    formContainer.appendChild(updateElementButton);
    formContainer.appendChild(cancelButton);
    formContainer.appendChild(showAllButton);
    formContainer.appendChild(updatePageButton);

    // Append form container to the overlay
    overlay.appendChild(formContainer);

    // Append overlay to the document body
    document.body.appendChild(overlay);

    // Cancel button click event: remove the overlay
    cancelButton.addEventListener('click', function () {
        document.body.removeChild(overlay);
    });

    showAllButton.addEventListener('click', function () {

        // get page name from url (ie: /login)
        page = window.location.pathname;
        updatePageButton.style.display = 'block';
        showAllButton.style.display = 'none';
        updateElementButton.style.display = 'none';

        fetch(`/preview/page${page}`)
            .then(response => response.json())
            .then(data => {
                console.log(data);
                // Display the edit form overlay with the retrieved JSON data
               // set inner text to data
                textarea.value = JSON.stringify(data, null, 2);
                // hide show all button
               
                // show update page button
                
            })
            .catch(error => {
                console.error('Error fetching preview element:', error);
            });

    });

    // Update button click event: parse the JSON and POST the updated data
    updateElementButton.addEventListener('click', function () {
        var updatedData;
        try {
            updatedData = JSON.parse(textarea.value);
        } catch (e) {
            alert('Invalid JSON. Please fix errors before updating.');
            return;
        }

        fetch(`/preview/element/${pid}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(updatedData)
        })
            .then(response => response.json())
            .then(result => {
                console.log('Update result:', result);
                // Optionally, you could display a success message here
                document.body.removeChild(overlay);
                // reload page
                location.reload();
            })
            .catch(error => {
                console.error('Error updating preview element:', error);
            });
    });

    updatePageButton.addEventListener('click', function () {
        page = window.location.pathname;
        var updatedData;
        try {
            updatedData = JSON.parse(textarea.value);
        } catch (e) {
            alert('Invalid JSON. Please fix errors before updating.');
            return;
        }

        fetch(`/preview/page${page}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(updatedData)
        })
            .then(response => response.json())
            .then(result => {
                console.log('Update page result:', result);
                // Optionally, you could display a success message here
                document.body.removeChild(overlay);
                // reload page
                location.reload();
            })
            .catch(error => {
                console.error('Error updating page element:', error);
            });
    });
}
