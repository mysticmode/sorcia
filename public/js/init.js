window.Sorcia = window.Sorcia || {};
Sorcia.toggleOverlay = function(state) {
	Sorcia._overlayEl = Sorcia._overlayEl || document.getElementById('overlay');
	if (typeof state === "undefined")
		state = false;
	if (!Sorcia._overlayEl)
		return;
	Sorcia._overlayEl.style.display = state ? "block" : "none";
}

window.addEventListener('pageshow', function() {
	Sorcia.toggleOverlay(false);
});
