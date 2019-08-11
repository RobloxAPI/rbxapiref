"use strict";
{
const settings = [
	{
		"name"     : "Theme",
		"type"     : "radio",
		"default"  : "Light",
		"options"  : [
			{"text": "Light", "value": "Light"},
			{"text": "Dark",  "value": "Dark"},
		],
	},
	{
		"name"     : "SecurityIdentity",
		"type"     : "select",
		"default"  : "0",
		"text"     : "Permission",
		"options"  : [
			{"text": "All",          "value": "0"},
			{"text": "Server",       "value": "7"},
			{"text": "CoreScript",   "value": "4"},
			{"text": "Command",      "value": "5"},
			{"text": "Plugin",       "value": "6"},
			{"text": "RobloxScript", "value": "3"},
			{"text": "Script",       "value": "2"},
		],
	},
	{
		"name"    : "ExpandMembers",
		"type"    : "checkbox",
		"default" : false,
		"text"    : "Expand all members",
	},
	{
		"name"     : "ShowDeprecated",
		"type"     : "checkbox",
		"default"  : true,
		"text"     : "Show deprecated",
	},
	{
		"name"     : "ShowNotBrowsable",
		"type"     : "checkbox",
		"default"  : true,
		"text"     : "Show unbrowsable",
	},
	{
		"name"     : "ShowHidden",
		"type"     : "checkbox",
		"default"  : true,
		"text"     : "Show hidden",
	}
];

function generateMenu(parent, settings, changed) {
	const idPrefix = "setting-";
	for (let setting of settings) {
		let value = window.localStorage.getItem(setting.name);
		if (value === null) {
			value = setting.default;
		};
		let section = document.createElement("div");
		section.className = setting.type;
		if (setting.type === "checkbox") {
			value = value === true || value === "true";
			let input = document.createElement("input");
			input.type = "checkbox";
			input.id = idPrefix + setting.name;
			input.name = setting.name;
			input.disabled = setting.disabled;
			input.defaultChecked = value;
			// Fires on toggle.
			input.addEventListener("change", function(event) {
				changed(event.target.name, event.target.checked, false);
			});

			let label = document.createElement("label");
			label.htmlFor = input.id;
			label.textContent = setting.text;

			section.appendChild(input);
			section.appendChild(label);
		} else if (setting.type === "radio") {
			for (let option of setting.options) {
				let input = document.createElement("input");
				input.type = "radio";
				input.id = idPrefix + setting.name + "-" + option.value;
				input.name = setting.name;
				input.value = option.value;
				input.disabled = setting.disabled || option.disabled;
				input.defaultChecked = value === option.value;
				// Fires on checked.
				input.addEventListener("change", function(event) {
					changed(event.target.name, event.target.value, false);
				});

				let label = document.createElement("label");
				label.htmlFor = input.id;
				label.textContent = option.text;

				section.appendChild(input);
				section.appendChild(label);
			};
		} else if (setting.type === "select") {
			let select = document.createElement("select");
			select.id = idPrefix + setting.name;
			select.disabled = setting.disabled;
			for (let option of setting.options) {
				let element = document.createElement("option");
				element.value = option.value;
				element.text = option.text;
				element.disabled = setting.disabled || option.disabled;
				element.defaultSelected = value === option.value;
				select.appendChild(element);
			};
			// Fires on select.
			select.addEventListener("change", function(event) {
				// Unknown support for HTMLSelectElement.name.
				changed(setting.name, event.target.value, false);
			});

			let label = document.createElement("label");
			label.htmlFor = select.id;
			label.textContent = setting.text;

			section.appendChild(select);
			section.appendChild(label);
		};
		parent.appendChild(section);
	};
};

class Settings {
	constructor() {
		this.settings = new Map();
	};
	Listen(name, listener) {
		let setting = this.settings.get(name);
		if (setting === undefined) {
			throw "unknown setting " + name;
		};
		if (typeof(listener) !== "function") {
			throw "listener must be a function";
		};
		setting.listeners.push(listener);

		let value = window.localStorage.getItem(name);
		if (value === null) {
			value = setting.config.default;
		};
		if (setting.config.type === "checkbox") {
			value = value === true || value === "true";
		};
		listener(name, value, true);
	};
	Changed(name, value, initial) {
		let setting = this.settings.get(name);
		if (setting === undefined) {
			return;
		};
		window.localStorage.setItem(name, value);
		for (let listener of setting.listeners) {
			listener(name, value, initial);
		};
	};
}

function initSettings() {
	let container = document.getElementById("main-header-right");
	if (container === null) {
		return;
	};
	container.insertAdjacentHTML('beforeend', '<div id="settings-button" class="header-block"><svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 14 14" height="14" width="14"><path d="M 6,0 5,2 3,1 1,3 2,5 0,6 v 2 l 2,1 -1,2 2,2 2,-1 1,2 h 2 l 1,-2 2,1 2,-2 -1,-2 2,-1 V 6 L 12,5 13,3 11,1 9,2 8,0 Z M 7,4 9,5 10,7 9,9 7,10 5,9 4,7 5,5 Z"/></svg></div>');
	container.insertAdjacentHTML('beforeend', '<form id="settings-menu" style="display:none;"></form>');

	let button = document.getElementById("settings-button");
	if (button === null) {
		return;
	};
	let menu = document.getElementById("settings-menu");
	if (menu === null) {
		return;
	};

	let rbxapiSettings = new Settings();
	generateMenu(menu, settings, function(name, value, initial) {
		rbxapiSettings.Changed(name, value, initial)
	});
	for (let setting of settings) {
		rbxapiSettings.settings.set(setting.name, {
			"config": setting,
			"listeners": [],
		});
		if (setting.disabled) {
			continue;
		};
		if (window.localStorage.getItem(setting.name) === null) {
			window.localStorage.setItem(setting.name, setting.default);
		};
	};

	button.addEventListener("click", function(event) {
		menu.style.display = "block";
		const onClick = function(event) {
			if (!menu.contains(event.target) && menu.style.display !== "none") {
				menu.style.display = "none";
				document.removeEventListener("click", onClick, true);
				event.preventDefault();
				event.stopPropagation();
			};
		};
		document.addEventListener("click", onClick, true);
		event.stopPropagation();
	});

	window.rbxapiSettings = rbxapiSettings;
	window.dispatchEvent(new Event("rbxapiSettings"));
};

if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", initSettings);
} else {
	initSettings();
};
};
