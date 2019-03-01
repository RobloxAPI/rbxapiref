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

function initSettingListeners() {
	let head = document.head;

	let showDeprecated = document.createElement("style");
	showDeprecated.innerHTML =
		".api-deprecated { display: none; }\n" +
		"#class-list .api-deprecated { display: unset; }\n" +
		"#class-list .api-deprecated > .element-link { display: none; }\n" +
		"#class-list .api-deprecated > ul { padding-left:0; border-left:none; }\n";
	window.rbxapiSettings.Listen("ShowDeprecated", function(name, value, initial) {
		if (value) {
			showDeprecated.remove();
		} else {
			head.appendChild(showDeprecated);
		};
	});

	let showNotBrowsable = document.createElement("style");
	showNotBrowsable.innerHTML =
		".api-not-browsable { display: none; }\n" +
		"#class-list .api-not-browsable { display: unset; }\n" +
		"#class-list .api-not-browsable > .element-link { display: none; }\n" +
		"#class-list .api-not-browsable > ul { padding-left:0; border-left:none; }\n";
	window.rbxapiSettings.Listen("ShowNotBrowsable", function(name, value, initial) {
		if (value) {
			showNotBrowsable.remove();
		} else {
			head.appendChild(showNotBrowsable);
		};
	});

	let showHidden = document.createElement("style");
	showHidden.innerHTML =
		".api-hidden { display: none; }\n" +
		"#class-list .api-hidden { display: unset; }\n" +
		"#class-list .api-hidden > .element-link { display: none; }\n" +
		"#class-list .api-hidden > ul { padding-left:0; border-left:none; }\n";
	window.rbxapiSettings.Listen("ShowHidden", function(name, value, initial) {
		if (value) {
			showHidden.remove();
		} else {
			head.appendChild(showHidden);
		};
	});
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

if (window.rbxapiSettings) {
	initSettingListeners();
} else {
	window.addEventListener("rbxapiSettings", initSettingListeners);
};
};
