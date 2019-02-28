"use strict";
{
function initTopNav() {
	let topnav = document.getElementById("top-nav");
	if (topnav === null) {
		return;
	};
	function updateTopNav() {
		topnav.style.visibility = window.pageYOffset === 0 ? "hidden" : "visible";
	};
	window.addEventListener("scroll", updateTopNav);
	updateTopNav();
};

function initHistoryToggle() {
	function toggleAll(show) {
		for (let item of document.querySelectorAll("#history > .patch-list > li")) {
			let diffElement = item.attributes["diff-element"];
			if (diffElement === undefined || diffElement.value === "Class" || diffElement.value === "Enum") {
				continue;
			};
			if (show) {
				item.style.display = "none";
			} else {
				item.style.display = "";
			};
		};
	};

	let controls = document.getElementById("history-controls");
	if (controls !== null) {
		controls.insertAdjacentHTML("beforeend", '<label><input type="checkbox" id="hide-member-changes">Hide member changes</label>');
	};

	let hideMemberChanges = document.getElementById("hide-member-changes");
	if (hideMemberChanges !== null) {
		hideMemberChanges.addEventListener("click", function(event) {
			toggleAll(event.target.checked, false);
		});
		toggleAll(hideMemberChanges.checked, true);
	} else {;
		toggleAll(false, true);
	};
};

if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", function() {
		initTopNav();
		initHistoryToggle();
	});
} else {
	initTopNav();
	initHistoryToggle();
};
};
