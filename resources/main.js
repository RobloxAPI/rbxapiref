"use strict";
const securityIdentities = [
	"All",
	"Server",
	"CoreScript",
	"BuiltinPlugin",
	"Command",
	"Plugin",
	"Script",
];
const securityPermissions = new Map([
	//                          ALL SVR CSC BPL CMD PLG SCR
	["None"                  , [ 1 , 1 , 1 , 1 , 1 , 1 , 1 ]],
	["RobloxPlaceSecurity"   , [ 1 , 1 , 1 , 1 , 1 , 1 , 0 ]],
	["PluginSecurity"        , [ 1 , 1 , 1 , 1 , 1 , 1 , 0 ]],
	["LocalUserSecurity"     , [ 1 , 1 , 1 , 0 , 1 , 0 , 0 ]],
	["RobloxScriptSecurity"  , [ 1 , 1 , 1 , 1 , 0 , 0 , 0 ]],
	["RobloxSecurity"        , [ 1 , 1 , 0 , 0 , 0 , 0 , 0 ]],
	["NotAccessibleSecurity" , [ 1 , 0 , 0 , 0 , 0 , 0 , 0 ]],
]);
const securityContexts = Array.from(securityPermissions.keys());
function secIDHasContext(id, ctx) {
	if (ctx instanceof Array) {
		// Return true if any context returns true.
		for (let c of ctx) {
			if (secIDHasContext(id, c)) {
				return true;
			};
		};
		return false;
	} else if (typeof(ctx) === "number") {
		ctx = securityContexts[ctx];
		if (!ctx) {
			return false;
		};
	};
	if (typeof(id) === "string") {
		id = securityIdentities.indexOf(id);
		if (id < 0) {
			return false;
		};
	};
	return securityPermissions.get(ctx)[id] === 1;
};
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

	let showRemoved = document.createElement("style");
	showRemoved.innerHTML =
		".api-removed { display: none; }\n" +
		"#class-list .api-removed { display: unset; }\n" +
		"#class-list .api-removed > .element-link { display: none; }\n" +
		"#class-list .api-removed > ul { padding-left:0; border-left:none; }\n";
	window.rbxapiSettings.Listen("ShowRemoved", function(name, value, initial) {
		if (value) {
			showRemoved.remove();
		} else {
			head.appendChild(showRemoved);
		};
		rbxapiActions.UpdateAll();
	});

	let security = new Map();
	for (let i = 0; i < securityIdentities.length; i++) {
		let content = "";
		for (let primary of securityPermissions) {
			if (primary[1][i] !== 0) {
				continue;
			};
			content += ".api-sec-" + primary[0];
			for (let secondary of securityPermissions) {
				if (secondary[1][i] !== 1) {
					continue;
				};
				content += ":not(.api-sec-" + secondary[0] + ")";
			};
			content += ",\n";
		};
		if (content === "") {
			continue;
		};
		content = content.slice(0, -2) + " {\n\tdisplay: none;\n}\n";
		let style = document.createElement("style");
		style.innerHTML = content;
		security.set(securityIdentities[i], style);
		console.log("CHECK", securityIdentities[i]);
		console.log(content);
		console.log("---------------------------------------------------");
	};
	window.rbxapiSettings.Listen("SecurityIdentity", function(name, value, initial) {
		for (let entry of security) {
			if (value === entry[0]) {
				head.appendChild(entry[1]);
			} else {
				entry[1].remove();
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

if (document.readyState === "completed") {
	initLoad();
} else {
	window.addEventListener("load", initLoad);
};

};
