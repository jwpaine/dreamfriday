const BASE_URL = window.location.origin; // Gets the current base URL


async function manage() {
   
        const response = await fetch(`${BASE_URL}/preview/json`);
        const data = await response.json();
        console.log(data);

} 

