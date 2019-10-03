"use strict";
{
function clearList(list) {
	while (list.lastChild) {
		list.removeChild(list.lastChild);
	};
};

function sortByTree(list, classes, parents) {
	clearList(list);
	for (let item of parents) {
		item[1].appendChild(item[0]);
	};
};

function sortByName(list, classes, parents) {
	clearList(list);
	for (let item of classes) {
		list.appendChild(item[0]);
	};
};

function initSortClasses() {
	let list = document.getElementById("class-list");
	if (list === null) {
		return;
	};
	let classes = [];
	let parents = [];
	for (let li of list.querySelectorAll("li")) {
		classes.push([li, li.querySelector(".element-link").text]);
		parents.push([li, li.parentNode]);
	};
	classes.sort(function(a, b) {
		return a[1].localeCompare(b[1]);
	});

	let methods = [
		[sortByTree, "Tree", true],
		[sortByName, "Name"]
	];

	const storageKey = "ClassSort"
	let defaultSort = window.localStorage.getItem(storageKey);
	if (defaultSort !== null) {
		for (let method of methods) {
			if (method[1] == defaultSort) {
				for (let method of methods) {
					method[2] = method[1] == defaultSort;
				};
				break;
			};
		};
	};

	let controls = document.createElement("div");
	controls.className = "class-list-controls";
	list.insertAdjacentElement("beforebegin", controls);
	for (let method of methods) {
		let input = document.createElement("input");
		input.type = "radio";
		input.id = "class-sort-" + method[1];
		input.name = "sort";
		input.value = method[1];
		input.checked = method[2];
		controls.appendChild(input);
		let label = document.createElement("label");
		label.htmlFor = input.id;
		label.appendChild(document.createTextNode(method[1]));
		controls.appendChild(label);
		let update = function(event) {
			method[0](list, classes, parents);
			window.localStorage.setItem(storageKey, method[1]);
		}
		input.addEventListener("click", update);
		if (method[2]) {
			update();
		};
	};
};

function domLoaded() {
	function formatCount(c) {
		return "(" + c + ")";
	};
	rbxapiActions.QuickLink(
		"#classes > header .element-count",
		"#class-list",
		["Count", "li > .element-link", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-classes > header .element-count",
		"#removed-class-list",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-classes",
		"#removed-class-list",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#enums > header .element-count",
		"#enum-list",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-enums > header .element-count",
		"#removed-enum-list",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-enums",
		"#removed-enum-list",
		["HideIfZero", ">*"]
	);
	rbxapiActions.QuickLink(
		"#removed-types > header .element-count",
		"#removed-type-list",
		["Count", ">*", formatCount]
	);
	rbxapiActions.QuickLink(
		"#removed-types",
		"#removed-type-list",
		["HideIfZero", ">*"]
	);

	initSortClasses();
};

function actionsLoaded() {
	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", domLoaded);
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
