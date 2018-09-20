"<div class=\"loading-message\"><div class=\"loading-spinner\"></div>Loading...</div>"

function expandMemberList(event) {
	event.preventDefault();
	var parent = event.target.closest(".inherited-members");
	if (parent === null) {
		return;
	};

	var list = parent.querySelector(".member-list");
	if (list !== null) {
		// Toggle list visibility.
		if (list.classList.contains("hidden")) {
			list.classList.remove("hidden");
		} else {
			list.classList.add("hidden");
		};
		return;
	}

	// Fetch the list.
	if (parent.classList.contains("loading")) {
		return;
	};
	parent.classList.add("loading");

	var link = parent.querySelector("a.element-link");
	if (link === null) {
		return;
	};
	parent.insertAdjacentHTML("beforeend", '<div class="loading-message"><div class="loading-spinner"></div>Loading...</div>');
	function clearLoader(event) {
		parent.classList.remove("loading");
		var loader = parent.querySelector(".loading-message");
		if (loader === null) {
			return;
		};
		parent.removeChild(loader);
	};
	function onLoaded(event) {
		clearLoader(event);
		if (event.target.response === null) {
			return;
		};
		var list = parent.querySelector(".member-list");
		if (list !== null) {
			return;
		};
		list = event.target.response.querySelector(".member-list");
		if (list === null) {
			return;
		};
		parent.insertBefore(list, null);
	};

	var req = new XMLHttpRequest();
	req.addEventListener("load", onLoaded);
	req.addEventListener("error", clearLoader);
	req.addEventListener("abort", clearLoader);
	req.open("GET", link.href);
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
