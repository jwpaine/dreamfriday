
async function loginWithEth() {
    if (!window.ethereum) {
        alert("Ethereum provider not detected. Please install MetaMask.");
        return;
    }

    try {
        // Request Ethereum account
        const accounts = await ethereum.request({ method: "eth_requestAccounts" });
        const address = accounts[0];

        // Get login challenge from server
        const response = await fetch("/auth/request?address=" + address);
        const { challenge } = await response.json();

        // Sign the challenge using MetaMask
        const signature = await ethereum.request({
            method: "personal_sign",
            params: [challenge, address],
        });

        // Send signed message back to the server
        const verifyResponse = await fetch("/auth/callback", {
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
        alert("Error logging in through wallet p");
    }
}
