"use strict";
{
function domLoaded() {
	// ToC
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
		"#toc-referrers",
		"#referrers > ul",
		["HideIfZero", ">*"]
	);

	// Sections
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
		"#referrers",
		"#referrers > ul",
		["HideIfZero", ">*"]
	);

	// Counters
	function formatCount(c) {
		return "(" + c + ")";
	};
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
		"#referrers > header .element-count",
		"#referrers > ul",
		["Count", ">*", formatCount]
	);
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
