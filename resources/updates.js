"use strict";
{
function toggleList(event) {
	let parent = event.target.closest(".update");
	if (parent === null) {
		return;
	};
	let list = parent.querySelector(".patch-list");
	if (list === null) {
		return;
	};
	if (list.style.display === "none") {
		list.style.display = "";
	} else {
		list.style.display = "none";
	};
};

function toggleAll(show, scroll) {
	let scrollTo;
	for (let item of document.querySelectorAll("#update-list > li .patch-list")) {
		let anchor = item.parentElement.querySelector(":target");
		if (anchor !== null) {
			scrollTo = anchor;
		}
		if (show) {
			item.style.display = "";
		} else {
			if (anchor !== null) {
				item.style.display = "";
			} else {
				item.style.display = "none";
			};
		};
	};
	if (scroll && scrollTo !== undefined) {
		scrollTo.scrollIntoView(true);
	};
};

function initUpdates() {
	// Inject pointer style.
	function initStyle() {
		let style = document.getElementById("updates-style");
		if (style !== null) {
			try {
				style.sheet.insertRule(".patch-list-toggle {cursor: pointer;}");
			} catch (error) {
			};
		};
	};
	if (document.readyState === "complete") {
		initStyle();
	} else {
		window.addEventListener("load", initStyle);
	};

	// Insert update controls.
	let controls = document.getElementById("update-controls");
	if (controls !== null) {
		controls.insertAdjacentHTML("beforeend", '<label><input type="checkbox" id="expand-all">Show all changes</label>');
	};

	// Init visibility toggle.
	for (let item of document.querySelectorAll("#update-list > li .patch-list-toggle")) {
		item.addEventListener("click", toggleList);
	};

	// Insert instructions.
	let list = document.getElementById("update-list");
	if (list !== null) {
		let note = document.createElement("div");
		note.innerText = "Click a date to expand or collapse changes.";
		list.parentElement.insertBefore(note, list);
	};

	// Init expand-all control.
	let expandAll = document.getElementById("expand-all");
	if (expandAll !== null) {
		expandAll.addEventListener("click", function(event) {
			toggleAll(event.target.checked, false);
		});
		toggleAll(expandAll.checked, true);
	} else {;
		toggleAll(false, true);
	};

	// Scroll to targeted patch item.
	let targetID = document.location.hash.slice(1);
	if (targetID !== "") {
		let target = document.getElementById(targetID);
		if (target) {
			if (target.parentElement.matches(".patch-list")) {
				target.parentElement.style.display = "";
				// TODO: The browser should automatically scroll to the target
				// at some point, but this might race.

				// Enabling scrollIntoView cancels the automatic scroll by the
				// browser, but then misses the target. Probably because the
				// scroll position is set before the list expansion is rendered.

				// target.scrollIntoView(true);
				return;
			};
		};
	};

	// No specific update is being targeted; expand latest updates.
	for (let update of document.querySelectorAll("#update-list .update")) {
		let list = update.querySelector(".patch-list");
		if (list === null) {
			continue;
		};
		list.style.display = "";
		// Expand up to first non-empty update.
		if (list.querySelector(".no-changes") === null) {
			break;
		};
	};
};

if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", initUpdates);
} else {
	initUpdates();
};
};
