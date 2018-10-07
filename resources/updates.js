"use strict";

function toggleList(event) {
	let parent = event.target.closest(".update");
	if (parent === null) {
		return;
	};
	let list = parent.querySelector(".patch-list");
	if (list === null) {
		return;
	};
	if (list.classList.contains("hidden")) {
		list.classList.remove("hidden");
	} else {
		list.classList.add("hidden");
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
			item.classList.remove("hidden");
		} else {
			if (anchor !== null) {
				item.classList.remove("hidden");
			} else {
				item.classList.add("hidden");
			};
		};
	};
	if (scroll && scrollTo !== undefined) {
		scrollTo.scrollIntoView(true);
	};
};

function parseDate(t) {
	if (t === null) {
		return null;
	};
	let datetime = t.attributes.datetime;
	if (datetime === undefined) {
		return null;
	};
	let p = datetime.value.match(/^(\d\d\d\d)\-(\d\d)\-(\d\d) (\d\d):(\d\d):(\d\d)/);
	if (p === null) {
		return null;
	};
	return new Date(p[1], p[2]-1, p[3], p[4], p[5], p[6]);
};

document.addEventListener("DOMContentLoaded", function(event) {
	let controls = document.getElementById("update-controls");
	if (controls !== null) {
		controls.insertAdjacentHTML("beforeend", '<label><input type="checkbox" id="expand-all">Show all changes</label>');
	};

	let expandAll = document.getElementById("expand-all");
	if (expandAll !== null) {
		expandAll.addEventListener("click", function(event) {
			toggleAll(event.target.checked, false);
		});
		toggleAll(expandAll.checked, true);
	} else {;
		toggleAll(false, true);
	};

	for (let item of document.querySelectorAll("#update-list > li .patch-list-toggle")) {
		item.addEventListener("click", toggleList);
	};

	let list = document.getElementById("update-list");
	if (list !== null) {
		let note = document.createElement("div");
		note.innerText = "Click a date to expand or collapse changes.";
		list.parentElement.insertBefore(note, list);
	};

	let style = document.getElementById("updates-style");
	if (style !== null) {
		try {
			style.sheet.insertRule(".patch-list-toggle {cursor: pointer;}");
		} catch (error) {
		};
	};

	if (!document.querySelector(".update :target")) {
		// No specific update is being targeted; expand latest updates.
		let day = 86400000; // ms
		let latest = null;
		for (let update of document.querySelectorAll("#update-list .update")) {
			let list = update.querySelector(".patch-list");
			if (list === null) {
				continue;
			};
			let date = parseDate(update.querySelector("time"));
			if (date === null) {
				continue;
			};
			if (latest === null) {
				latest = date;
			};
			if (latest.getTime() - date.getTime() <= day) {
				list.classList.remove("hidden");
			};
		};
	};
});
