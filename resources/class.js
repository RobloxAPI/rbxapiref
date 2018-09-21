"use strict";

function expandMemberList(event) {
	// Prevent clicked anchor from doing anything else.
	event.preventDefault();

	let parent = event.target.closest(".inherited-members");
	if (parent === null) {
		return;
	};

	// Attempt to toggle a list that was loaded previously.
	let list = parent.querySelector("#members .member-list");
	if (list !== null) {
		if (list.classList.contains("hidden")) {
			list.classList.remove("hidden");
		} else {
			list.classList.add("hidden");
		};
		return;
	}

	let link = parent.querySelector("a.element-link");
	if (link === null || link.href.length === 0) {
		return;
	};
	let url = link.href

	// Create a message indicating that data is being loaded. Also use this
	// message as a "lock", which will usually prevent multiple requests at
	// once by this element.
	if (parent.querySelector(".loading-message") !== null) {
		return;
	};
	parent.insertAdjacentHTML("beforeend", '<div class="loading-message"><div class="loading-spinner"></div>Loading...</div>');

	function clearLoader(event) {
		let loader = parent.querySelector(".loading-message");
		if (loader === null) {
			return;
		};
		parent.removeChild(loader);
	};

	function onLoaded(event) {
		if (event.target.response === null) {
			return;
		};
		let list = parent.querySelector(".member-list");
		if (list !== null) {
			return;
		};
		list = event.target.response.querySelector("#members .member-list");
		if (list === null) {
			return;
		};
		parent.insertBefore(list, null);
	};

	let req = new XMLHttpRequest();
	req.addEventListener("load", function(event) {
		onLoaded(event);
		clearLoader(event);
	});
	req.addEventListener("error", clearLoader);
	req.addEventListener("abort", clearLoader);
	req.open("GET", url);
	req.responseType = "document";
	req.send();
};

document.addEventListener("DOMContentLoaded", function(event) {
	for (parent of document.getElementsByClassName("inherited-members")) {
		var count = parent.querySelector("a.member-count");
		var link = parent.querySelector("a.element-link");
		if (count === null || link === null) {
			return;
		};
		count.href = "#";
		count.title = "Click to expand inherited members.";
		count.addEventListener("click", expandMemberList);
	};
});
