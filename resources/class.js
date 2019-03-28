"use strict";
{
function expandMemberList(element, force) {
	let placehold = element.closest(".inherited-members");
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
		if (typeof(force) === "boolean") {
			if (force) {
				body.style.display = "";
			} else {
				body.style.display = "none";
			};
			rbxapiActions.Update(head, element);
			return;
		};
		if (body.style.display === "none") {
			body.style.display = "";
		} else {
			body.style.display = "none";
		};
		rbxapiActions.Update(head, element);
		return;
	}

	if (force === false) {
		return;
	};

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

	function formatSingular(c) {
		return c + " member";
	};
	function formatPlural(c) {
		return c + " members";
	};
	function onLoaded(event) {
		if (event.target.response === null) {
			return;
		};
		let body = nextMatching(head, "tbody.inherited-members-list", "thead");
		if (body !== null) {
			return;
		};
		body = event.target.response.querySelector("#members-index .index-card tbody");
		if (body === null) {
			return;
		};
		body.classList.add("inherited-members-list");
		head.insertAdjacentElement("afterend", body);
		window.rbxapiActions.Link(head, false, ["HideIfZero", body, ">*"])
		window.rbxapiActions.Link(element, false, ["Count", body, ">*", formatSingular, formatPlural]);
		rbxapiActions.Update(head, element);
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

function settingsLoaded() {
	window.rbxapiSettings.Listen("ExpandMembers", function(name, value, initial) {
		for (let count of document.querySelectorAll(".inherited-members a.member-count")) {
			expandMemberList(count, value);
		};
	});
};

function domLoaded() {
	for (let parent of document.getElementsByClassName("inherited-members")) {
		let count = parent.querySelector("a.member-count");
		if (count === null) {
			continue;
		};
		count.href = "#";
		count.title = "Click to toggle visibility of members.";
		count.addEventListener("click", function(event) {
			// Prevent clicked anchor from doing anything else.
			event.preventDefault();
			expandMemberList(event.target);
		});
	};

	// ToC
	rbxapiActions.QuickLink(
		"#toc-superclasses",
		"#superclasses > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-subclasses",
		"#subclasses > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-class-tree",
		"#toc-class-tree > ol",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-removed-members-index",
		"#removed-members-index > .index-card > tbody:first-of-type",
		["HideIfZero", ">:not(.empty)"]
	);
	rbxapiActions.QuickLink(
		"#toc-members",
		"#member-sections",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-removed-members",
		"#removed-member-sections",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-classes",
		"#classes > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-enums",
		"#enums > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-referrers",
		"#referrers > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#toc-references",
		"#toc-references > ol",
		["HideIfZero", ">*"]
	);

	// Sections
	rbxapiActions.QuickLink(
		"#superclasses",
		"#superclasses > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#subclasses",
		"#subclasses > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#tree",
		"#tree",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#removed-members-index",
		"#removed-members-index > .index-card > tbody:first-of-type",
		["HideIfZero", ">:not(.empty)"]
	);
	rbxapiActions.QuickLink(
		"#members",
		"#member-sections",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#removed-members",
		"#removed-member-sections",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#classes",
		"#classes > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#enums",
		"#enums > ul",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#referrers",
		"#referrers > ul",
		["HideIfZero", ">*"]
	);

	// Counters
	function formatCount(c) {
		return "(" + c + ")";
	};
	rbxapiActions.QuickLink(
		"#superclasses > header .element-count",
		"#superclasses > ul",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#subclasses > header .element-count",
		"#subclasses > ul",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#members-index > header .element-count",
		"#members-index > .index-card > tbody:first-of-type",
		["Count", ">:not(.empty)", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-members-index > header .element-count",
		"#removed-members-index > .index-card > tbody:first-of-type",
		["Count", ">:not(.empty)", formatCount]
	);
	rbxapiActions.QuickLink(
		"#classes > header .element-count",
		"#classes > ul",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#enums > header .element-count",
		"#enums > ul",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#referrers > header .element-count",
		"#referrers > ul",
		["Count", ">*", formatCount]
	);

	if (window.rbxapiSettings) {
		settingsLoaded();
	} else {
		window.addEventListener("rbxapiSettings", settingsLoaded);
	};
};

function actionsLoaded() {
	if (document.readyState === "loading") {
		window.addEventListener("DOMContentLoaded", domLoaded);
	} else {
		domLoaded();
	};
};

if (window.rbxapiActions) {
	actionsLoaded();
} else {
	window.addEventListener("rbxapiActions", actionsLoaded);
};
};
