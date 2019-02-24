"use strict";

const settings = [
	{
		"name"     : "Theme",
		"type"     : "radio",
		"default"  : "Light",
		"disabled" : true,
		"options"  : [
			{"text": "Light", "value": "Light"},
			{"text": "Dark",  "value": "Dark"},
		],
	},
	{
		"name"     : "SecurityIdentity",
		"type"     : "select",
		"default"  : "0",
		"text"     : "Context",
		"disabled" : true,
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
		"disabled" : true,
	},
	{
		"name"     : "ShowBrowsable",
		"type"     : "checkbox",
		"default"  : true,
		"text"     : "Show browsable",
		"disabled" : true,
	},
	{
		"name"     : "ShowHidden",
		"type"     : "checkbox",
		"default"  : true,
		"text"     : "Show hidden",
		"disabled" : true,
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

let registeredSettings = new Map();
function settingChanged(name, value, initial) {
	let setting = registeredSettings[name];
	if (setting === undefined) {
		return;
	};
	window.localStorage.setItem(name, value);
	for (let listener of setting.listeners) {
		listener(name, value, initial);
	};
};
function RegisterSettingListener(name, listener) {
	let setting = registeredSettings[name];
	if (setting === undefined) {
		throw "unknown setting " + name;
		return;
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

document.addEventListener("DOMContentLoaded", function(event) {
	let container = document.getElementById("main-header-right");
	if (container === null) {
		return;
	};
	container.insertAdjacentHTML('beforeend', '<div id="settings-button" class="header-block"></div>');
	container.insertAdjacentHTML('beforeend', '<form id="settings-menu" style="display:none;"></form>');

	let button = document.getElementById("settings-button");
	if (button === null) {
		return;
	};
	let menu = document.getElementById("settings-menu");
	if (menu === null) {
		return;
	};

	generateMenu(menu, settings, settingChanged);
	for (let setting of settings) {
		registeredSettings[setting.name] = {
			"config": setting,
			"listeners": [],
		};
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

	document.dispatchEvent(new Event("rbxapiSettingsLoaded"));
});
