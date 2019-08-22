"use strict";
const securityIdentities = [7, 4, 5, 6, 3, 2];
const securityContexts = [
	"NotAccessibleSecurity",
	"RobloxSecurity",
	"RobloxScriptSecurity",
	"LocalUserSecurity",
	"PluginSecurity",
	"RobloxPlaceSecurity",
	"None",
];
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

let settingsLoaded = false;
function initSettings() {
	let head = document.head;

	window.rbxapiSettings.Listen("Theme", function(name, value, initial) {
		if (initial) {
			// Handled by quick-theme.js.
			return;
		};
		document.documentElement.className = value;
	});

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
		rbxapiActions.UpdateAll();
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
		rbxapiActions.UpdateAll();
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
		rbxapiActions.UpdateAll();
	});

	let security = [];
	for (let i=0; i<securityIdentities.length; i++) {
		let content = "";
		for (let c=0; c<=i; c++) {
			content += ".api-sec-" + securityContexts[c];
			for (let n=i+1; n<securityContexts.length; n++) {
				content += ":not(.api-sec-" + securityContexts[n] + ")";
			};
			if (c < i) {
				content += ", ";
			};
		};
		content += " { display: none; }\n";
		security[i] = document.createElement("style");
		security[i].innerHTML = content;
	};
	window.rbxapiSettings.Listen("SecurityIdentity", function(name, value, initial) {
		for (let i=0; i<security.length; i++) {
			if (Number(value) === securityIdentities[i]) {
				head.appendChild(security[i]);
			} else {
				security[i].remove();
			};
		};
		rbxapiActions.UpdateAll();
	});

	settingsLoaded = true;
	window.dispatchEvent(new Event("settingsLoaded"));
};

function initActions() {
	if (window.rbxapiSettings) {
		initSettings();
	} else {
		window.addEventListener("rbxapiSettings", initSettings);
	};
};

function fixTarget() {
	if (document.readState !== "completed") {
		let targetID = document.location.hash.slice(1);
		if (targetID !== "") {
			let target = document.getElementById(targetID);
			if (target) {
				target.scrollIntoView(true);
			};
		};
	};
};

function initLoad() {
	if (settingsLoaded) {
		fixTarget();
	} else {
		window.addEventListener("settingsLoaded", fixTarget);
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

if (window.rbxapiActions) {
	initActions();
} else {
	window.addEventListener("rbxapiActions", initActions);
};

if (document.readtState === "completed") {
	initLoad();
} else {
	window.addEventListener("load", initLoad);
};

};
