"use strict";
document.addEventListener("DOMContentLoaded", function() {
	let topnav = document.getElementById("top-nav");
	if (topnav === null) {
		return;
	};
	window.addEventListener("scroll", function(e) {
		topnav.style.visibility = e.pageY === 0 ? "hidden" : "visible";
	});
	topnav.style.visibility = window.pageYOffset === 0 ? "hidden" : "visible";
});
