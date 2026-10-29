if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("./service-worker.js").catch(() => {
        console.warn("Offline support could not be enabled.");
    });
}
