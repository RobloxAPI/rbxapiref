"use strict";

function expandMemberList(event) {
	// Prevent clicked anchor from doing anything else.
	event.preventDefault();

	let placehold = event.target.closest(".inherited-members");
	if (placehold === null) {
		return;
	};
	let head = placehold.closest("thead");
	if (head === null || !head.parentElement.classList.contains("index-card")) {
		return;
	};

	// Get the a subsequent sibling that matches query. Stop when a sibling
	// matching boundary is encountered.
	function nextMatching(element, query, boundary) {
		do {
			element = element.nextElementSibling;
			if (element === null) {
				break;
			};
			if (element.matches(query)) {
				return element;
			};
		} while (boundary && !element.matches(boundary));
		return null;
	};

	// Attempt to toggle a list that was loaded previously.
	let body = nextMatching(head, "tbody.inherited-members-list", "thead");
	if (body !== null) {
		if (body.style.display === "none") {
			body.style.display = "";
		} else {
			body.style.display = "none";
		};
		return;
	}

	let link = placehold.querySelector("a.element-link");
	if (link === null || link.href.length === 0) {
		return;
	};
	let url = link.href

	{
		// Create a message indicating that data is being loaded. Also use
		// this message as a "lock", which will usually prevent multiple
		// requests at once by this element.
		let loader = nextMatching(head, "tbody.loading-message", "thead")
		if (loader !== null) {
			return;
		};
		head.insertAdjacentHTML("afterend", '<tbody class="loading-message"><tr><td colspan=3><div class="loading-spinner"></div>Loading...</td></tr></tbody>');
	};

	function clearLoader(event) {
		let loader = nextMatching(head, "tbody.loading-message", "thead");
		if (loader === null) {
			return;
		};
		loader.parentElement.removeChild(loader);
	};

	function onLoaded(event) {
		if (event.target.response === null) {
			return;
		};
		let body = nextMatching(head, "tbody.inherited-members-list", "thead");
		if (body !== null) {
			return;
		};
		body = event.target.response.querySelector("#members .index-card tbody");
		if (body === null) {
			return;
		};
		body.classList.add("inherited-members-list");
		head.insertAdjacentElement("afterend", body);
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
		if (count === null) {
			continue;
		};
		count.href = "#";
		count.title = "Click to toggle visibility of members.";
		count.addEventListener("click", expandMemberList);
	};
});
