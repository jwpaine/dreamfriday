document.addEventListener("DOMContentLoaded", () => {
    const fadeElements = document.querySelectorAll(".fade");
    const fadeMap = {};

    // Store the fadeStart for each element in fadeMap
    fadeElements.forEach((element) => {
        const fadeStart = element.getBoundingClientRect().top;
        const classNameKey = element.className;
        fadeMap[classNameKey] = {
            element,
            fadeStart
        };
    });

    const fadeEnd = window.innerHeight * 0.75;

    window.addEventListener("scroll", () => {
        for (const key in fadeMap) {
            const { element, fadeStart } = fadeMap[key];
            const elementPosition = element.getBoundingClientRect().top;

            // Calculate the opacity based on scroll position
            let opacity = (elementPosition - fadeEnd) / (fadeStart - fadeEnd);
            opacity = Math.min(Math.max(opacity, 0), 1); // Clamp opacity between 0 and 1

            element.style.opacity = opacity;
        }
    });
});