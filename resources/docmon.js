"use strict";
{
function getNumber(cell) {
	return Number(cell.textContent);
};

function getString(cell) {
	return cell.textContent;
};

function getCell(cell) {
	return cell;
};

function getPercent(cell) {
	return Number(cell.textContent.slice(0, -1));
};

function presortNumber(i, j) {
	return this[i] - this[j];
};

function presortString(i, j) {
	if (this[i] < this[j]) {
		return -1;
	} else if (this[i] > this[j]) {
		return 1;
	};
	return 0;
};

function presortType(i, j) {
	if (this[i].textContent < this[j].textContent) {
		return -1;
	} else if (this[i].textContent > this[j].textContent) {
		return 1;
	};
	if (this[i].nextElementSibling.textContent < this[j].nextElementSibling.textContent) {
		return -1;
	} else if (this[i].nextElementSibling.textContent > this[j].nextElementSibling.textContent) {
		return 1;
	};
	return 0;
};

function presortSectionSummary(i, j) {
	if (this[i].className < this[j].className) {
		return 1;
	} else if (this[i].className > this[j].className) {
		return -1;
	};
	if (this[i].textContent < this[j].textContent) {
		return -1;
	} else if (this[i].textContent > this[j].textContent) {
		return 1;
	};
	return 0;
};

function presortSectionNumber(i, j) {
	if (this[i].className < this[j].className) {
		return 1;
	} else if (this[i].className > this[j].className) {
		return -1;
	};
	return Number(this[i].textContent) - Number(this[j].textContent);
};

function sortAsc(parent, rows, indexes) {
	for (let i = 0; i < indexes.length; i++) {
		parent.appendChild(rows[indexes[i]]);
	};
};

function sortDsc(parent, rows, indexes) {
	for (let i = indexes.length-1; i >= 0; i--) {
		parent.appendChild(rows[indexes[i]]);
	};
};

function initDocmon() {
	let coverage = document.querySelector("#coverage .value");
	if (coverage) {
		let value = Number(coverage.firstChild.data.slice(0, -1))/100;
		if (!Number.isNaN(value)) {
			let start, end;
			if (value >= 0.5) {
				start = "var(--theme-patch-change)"
				end = "var(--theme-patch-add)"
				value = value*2 - 1;
			} else {
				start = "var(--theme-patch-remove)"
				end = "var(--theme-patch-change)"
				value = value*2;
			};
			coverage.style.setProperty("--value", String(value));
			coverage.style.setProperty("--min-color", start);
			coverage.style.setProperty("--max-color", end);
		};
	};

	let cols = [
		[getNumber, presortNumber],
		[getCell, presortType],
		[getString, presortString],
		[getCell, presortSectionSummary],
		[getCell, presortSectionNumber],
		[getCell, presortSectionNumber],
		[getPercent, presortNumber],
	];

	let table = document.getElementById("status");
	if (!table) {
		return;
	};
	let head = table.querySelector("thead");
	if (!head) {
		return;
	};
	let body = table.querySelector("tbody");
	if (!body) {
		return;
	};

	let headers = head.rows[0].children;
	let rows = Array.prototype.slice.call(body.rows);
	let sorters = [];
	sorters.length = cols.length;
	for (let i = 0; i < sorters.length; i++) {
		let indexes = [];
		indexes.length = rows.length;
		let values = [];
		values.length = rows.length;
		for (let j = 0; j < values.length; j++) {
			indexes[j] = j;
			values[j] = cols[i][0](rows[j].children[i]);
		};
		indexes.sort(cols[i][1].bind(values));
		sorters[i] = indexes;

		headers[i].classList.add("sortable");
		headers[i].onclick = function(event) {
			let header = event.target;
			if (header.classList.contains("asc")) {
				for (let h of headers) {
					h.classList.remove("asc");
					h.classList.remove("dsc");
				};
				header.classList.add("dsc");
				sortDsc(body, rows, indexes);
			} else if (header.classList.contains("dsc")) {
				for (let h of headers) {
					h.classList.remove("asc");
					h.classList.remove("dsc");
				};
				sortAsc(body, rows, sorters[0]);
			} else {
				for (let h of headers) {
					h.classList.remove("asc");
					h.classList.remove("dsc");
				};
				header.classList.add("asc");
				sortAsc(body, rows, indexes);
			};
		};
	};
};


if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", initDocmon);
} else {
	initDocmon();
};
};
