"use strict";
{
function domLoaded() {
	// ToC
	rbxapiActions.QuickLink(
		"#toc-referrers",
		"#referrers > ul",
		["HideIfZero", ">*"]
	);

	// Sections
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
