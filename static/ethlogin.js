const BASE_URL = window.location.origin; // Gets the current base URL


async function loginWithEth() {
    if (!window.ethereum) {
        alert("Ethereum provider not detected. Please install MetaMask.");
        return;
    }

    try {
        const accounts = await ethereum.request({ method: "eth_requestAccounts" });
        const address = accounts[0];

        // ✅ Dynamically use the current base URL
        const response = await fetch(`${BASE_URL}/auth/request?address=${address}`);
        const { challenge } = await response.json();

        const signature = await ethereum.request({
            method: "personal_sign",
            params: [challenge, address],
        });

        const verifyResponse = await fetch(`${BASE_URL}/auth/callback`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ address, challenge, signature }),
        });

        const result = await verifyResponse.json();
        console.log(result);

        if (result.status === "accepted") {
            window.location.href = "/admin";
        }
    } catch (error) {
        console.error("MetaMask Login Error:", error);
        alert("Error logging in through wallet");
    }
}
