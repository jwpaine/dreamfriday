const BASE_URL = window.location.origin; // Gets the current base URL

let page = ""

function manage() {

   
    const messagesElement = document.getElementById('messages');

  
    getPreviewData(function(data, error) {
        if(error) {
            console.log(error);
            if (messagesElement) {
                messagesElement.textContent = `Error: ${error.message}`;
            }
            return;
        }
        console.log(data);
    })
}

async function getPreviewData(callback) {
    try {
        const response = await fetch(`${BASE_URL}/preview/json`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        callback(data, null);

    } catch (error) {
        console.error('Error fetching preview JSON:', error);
        callback(null, error);
    }

}




