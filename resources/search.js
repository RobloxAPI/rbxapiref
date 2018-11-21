"use strict";
const devhubBaseURL = "https://developer.roblox.com/api-reference";

/* BEGIN fts_fuzzy_match.js */
// LICENSE
//
//   This software is dual-licensed to the public domain and under the following
//   license: you are granted a perpetual, irrevocable license to copy, modify,
//   publish, and distribute this file as you see fit.
//
// VERSION
//   0.1.0  (2016-03-28)  Initial release
//
// AUTHOR
//   Forrest Smith
//
// CONTRIBUTORS
//   J?rgen Tjern? - async helper


// Returns true if each character in pattern is found sequentially within str
function fuzzy_match_simple(pattern, str) {

	var patternIdx = 0;
	var strIdx = 0;
	var patternLength = pattern.length;
	var strLength = str.length;

	while (patternIdx != patternLength && strIdx != strLength) {
		var patternChar = pattern.charAt(patternIdx).toLowerCase();
		var strChar = str.charAt(strIdx).toLowerCase();
		if (patternChar == strChar)
			++patternIdx;
		++strIdx;
	}

	return patternLength != 0 && strLength != 0 && patternIdx == patternLength ? true : false;
}

// Returns [bool, score, formattedStr]
// bool: true if each character in pattern is found sequentially within str
// score: integer; higher is better match. Value has no intrinsic meaning. Range varies with pattern.
//        Can only compare scores with same search pattern.
// formattedStr: input str with matched characters marked in <b> tags. Delete if unwanted.
function fuzzy_match(pattern, str) {

	// Score consts
	var adjacency_bonus = 5;                // bonus for adjacent matches
	var separator_bonus = 10;               // bonus if match occurs after a separator
	var camel_bonus = 10;                   // bonus if match is uppercase and prev is lower
	var leading_letter_penalty = -3;        // penalty applied for every letter in str before the first match
	var max_leading_letter_penalty = -9;    // maximum penalty for leading letters
	var unmatched_letter_penalty = -1;      // penalty for every letter that doesn't matter

	// Loop variables
	var score = 0;
	var patternIdx = 0;
	var patternLength = pattern.length;
	var strIdx = 0;
	var strLength = str.length;
	var prevMatched = false;
	var prevLower = false;
	var prevSeparator = true;       // true so if first letter match gets separator bonus

	// Use "best" matched letter if multiple string letters match the pattern
	var bestLetter = null;
	var bestLower = null;
	var bestLetterIdx = null;
	var bestLetterScore = 0;

	var matchedIndices = [];

	// Loop over strings
	while (strIdx != strLength) {
		var patternChar = patternIdx != patternLength ? pattern.charAt(patternIdx) : null;
		var strChar = str.charAt(strIdx);

		var patternLower = patternChar != null ? patternChar.toLowerCase() : null;
		var strLower = strChar.toLowerCase();
		var strUpper = strChar.toUpperCase();

		var nextMatch = patternChar && patternLower == strLower;
		var rematch = bestLetter && bestLower == strLower;

		var advanced = nextMatch && bestLetter;
		var patternRepeat = bestLetter && patternChar && bestLower == patternLower;
		if (advanced || patternRepeat) {
			score += bestLetterScore;
			matchedIndices.push(bestLetterIdx);
			bestLetter = null;
			bestLower = null;
			bestLetterIdx = null;
			bestLetterScore = 0;
		}

		if (nextMatch || rematch) {
			var newScore = 0;

			// Apply penalty for each letter before the first pattern match
			// Note: std::max because penalties are negative values. So max is smallest penalty.
			if (patternIdx == 0) {
				var penalty = Math.max(strIdx * leading_letter_penalty, max_leading_letter_penalty);
				score += penalty;
			}

			// Apply bonus for consecutive bonuses
			if (prevMatched)
				newScore += adjacency_bonus;

			// Apply bonus for matches after a separator
			if (prevSeparator)
				newScore += separator_bonus;

			// Apply bonus across camel case boundaries. Includes "clever" isLetter check.
			if (prevLower && strChar == strUpper && strLower != strUpper)
				newScore += camel_bonus;

			// Update patter index IFF the next pattern letter was matched
			if (nextMatch)
				++patternIdx;

			// Update best letter in str which may be for a "next" letter or a "rematch"
			if (newScore >= bestLetterScore) {

				// Apply penalty for now skipped letter
				if (bestLetter != null)
					score += unmatched_letter_penalty;

				bestLetter = strChar;
				bestLower = bestLetter.toLowerCase();
				bestLetterIdx = strIdx;
				bestLetterScore = newScore;
			}

			prevMatched = true;
		}
		else {
			// Append unmatch characters
			formattedStr += strChar;

			score += unmatched_letter_penalty;
			prevMatched = false;
		}

		// Includes "clever" isLetter check.
		prevLower = strChar == strLower && strLower != strUpper;
		prevSeparator = strChar == '_' || strChar == ' ';

		++strIdx;
	}

	// Apply score for last match
	if (bestLetter) {
		score += bestLetterScore;
		matchedIndices.push(bestLetterIdx);
	}

	// Finish out formatted string after last pattern matched
	// Build formated string based on matched letters
	var formattedStr = "";
	var lastIdx = 0;
	for (var i = 0; i < matchedIndices.length; ++i) {
		var idx = matchedIndices[i];
		formattedStr += str.substr(lastIdx, idx - lastIdx) + "<b>" + str.charAt(idx) + "</b>";
		lastIdx = idx + 1;
	}
	formattedStr += str.substr(lastIdx, str.length - lastIdx);

	var matched = patternIdx == patternLength;
	return [matched, score, formattedStr];
}

// Strictly optional utility to help make using fts_fuzzy_match easier for large data sets
// Uses setTimeout to process matches before a maximum amount of time before sleeping
//
// To use:
//      var asyncMatcher = new fts_fuzzy_match(fuzzy_match, "fts", "ForrestTheWoods",
//                                              function(results) { console.log(results); });
//      asyncMatcher.start();
//
function fts_fuzzy_match_async(pattern, database, onComplete) {
	var ITEMS_PER_CHECK = 1000;         // performance.now can be very slow depending on platform

	var max_ms_per_frame = 1000.0/30.0; // 30FPS
	var itemIndex = 0;
	var itemCount = database.itemCount;
	var itemOffset = database.OFFSET_STRINGS;
	var results = [];
	var resumeTimeout = null;

	// Perform matches for at most max_ms
	function step() {
		clearTimeout(resumeTimeout);
		resumeTimeout = null;

		var stopTime = performance.now() + max_ms_per_frame;
		while (itemIndex < itemCount && itemOffset < database.data.byteLength) {
			if ((itemIndex % ITEMS_PER_CHECK) == 0) {
				if (performance.now() > stopTime) {
					resumeTimeout = setTimeout(step, 1);
					return;
				}
			}

			let len = database.data.getUint8(itemOffset);
			itemOffset++;
			let item = new DatabaseItem(
				database.itemData(itemIndex),
				new TextDecoder().decode(database.data.buffer.slice(itemOffset, itemOffset + len))
			)
			if (item.dbType === "class") {
				item.iconIndex = database.icon(itemIndex);
			};
			itemOffset += len;
			itemIndex++;
			let split = item.name.indexOf(".");
			let prefix = "";
			let suffix = item.name;
			if (split > 0) {
				prefix = item.name.slice(0,split+1);
				suffix = item.name.slice(split+1);
			};
			let result = fuzzy_match(pattern, suffix);
			if (result[0] === true) {
				result[2] = prefix + result[2];
				results.push([result, item]);
			};
		}

		onComplete(results);
		return null;
	};

	// Abort current process
	this.cancel = function() {
		if (resumeTimeout !== null)
			clearTimeout(resumeTimeout);
	};

	// Must be called to start matching.
	// I tried to make asyncMatcher auto-start via "var resumeTimeout = step();"
	// However setTimout behaving in an unexpected fashion as onComplete insisted on triggering twice.
	this.start = function() {
		step();
	}

	// Process full list. Blocks script execution until complete
	this.flush = function() {
		max_ms_per_frame = Infinity;
		step();
	}
};
/* END fts_fuzzy_match.js */


function getbit(p, a) {
	return (p>>a)&1 === 1;
};
function getbits(p, a, b) {
	return (p >> a) & ((1<<(b-a)) - 1);
};

class DatabaseItem {
	constructor(data, string) {
		this.data = data;
		this.name = string;
		this.iconIndex = 0;
	};
	get removed() {
		return !!getbit(this.data, 3);
	};
	get deprecated() {
		return !!getbit(this.data, 4);
	};
	get protected() {
		return !!getbit(this.data, 5);
	};
	get dbType() {
		switch (getbits(this.data, 0, 3)) {
		case 0:
			return "class";
		case 1:
			return "enum";
		case 2:
			return "enumitem";
		case 3:
			return "type";
		case 4:
			return "property";
		case 5:
			return "function";
		case 6:
			return "event";
		case 7:
			return "callback";
		};
		return null;
	};
	get icon() {
		let icon = null;
		switch (getbits(this.data, 0, 3)) {
		case 0:
			icon = {class: "class-icon", index: this.iconIndex || 0};
			break;
		case 1:
			icon = {class: "enum-icon"};
			break;
		case 2:
			icon = {class: "enum-item-icon"};
			break;
		case 3:
			icon = {class: "member-icon", index: 3};
			break;
		case 4:
			icon = {class: "member-icon", index: 6};
			break;
		case 5:
			icon = {class: "member-icon", index: 4};
			break;
		case 6:
			icon = {class: "member-icon", index: 11};
			break;
		case 7:
			icon = {class: "member-icon", index: 16};
			break;
		};
		if (icon !== null && icon.index !== undefined && this.protected) {
			icon.index++;
		};
		return icon;
	};
}

class Database {
	constructor(data) {
		this.data = new DataView(data);
		this.SIZE_ICON = 1;
		this.SIZE_ITEM = 1;
		this.OFFSET_VERSION = 0;
		this.OFFSET_ICON_COUNT = 1;
		this.OFFSET_ITEM_COUNT = 3;
		this.OFFSET_ICONS = 5;
		this.OFFSET_ITEMS = this.OFFSET_ICONS + this.SIZE_ICON*this.iconCount;
		this.OFFSET_STRINGS = this.OFFSET_ITEMS + this.SIZE_ITEM*this.itemCount;
	};
	get version() {
		return this.data.getUint8(this.OFFSET_VERSION);
	};
	get iconCount() {
		return this.data.getUint16(this.OFFSET_ICON_COUNT, true);
	};
	get itemCount() {
		return this.data.getUint16(this.OFFSET_ITEM_COUNT, true);
	};
	icon(index) {
		index = index % this.iconCount;
		return this.data.getUint8(this.OFFSET_ICONS + this.SIZE_ICON*index);
	};
	itemData(index) {
		index = index % this.itemCount;
		return this.data.getUint16(this.OFFSET_ITEMS + this.SIZE_ITEM*index, true);
	};
	string(indices) {
		let single = false;
		if (typeof indices === "number") {
			indices = [indices];
			single = true;
		};
		let strings = new Array(indices.length);
		strings.fill(null);
		let filled = 0;
		let i = 0;
		let n = this.itemCount;
		let off = this.OFFSET_STRINGS;
		while (off < this.data.byteLength && i < n) {
			let len = this.data.getUint8(off);
			off++;
			for (let j = 0; j < indices.length; j++) {
				let index = indices[j] % this.itemCount;
				if (i === index) {
					strings[j] = new TextDecoder().decode(this.data.buffer.slice(off, off + len));
					filled++
					if (filled >= strings.length) {
						if (single) {
							return strings[0];
						};
						return strings;
					};
				};
			};
			off += len;
			i++;
		};
		if (single) {
			return strings[0];
		};
		return strings;
	};
	item(index) {
		index = index % this.data.getUint16(this.OFFSET_ITEM_COUNT, true);
		let item = new DatabaseItem(
			this.itemData(index),
			this.string(index)
		)
		if (item.dbType === "class") {
			item.iconIndex = this.icon(index);
		};
		return item;
	};
};

let database = null;
let fetchedDB = false;
function getDatabase(success, failure) {
	if (database === null) {
		if (fetchedDB) {
			// TODO: error message.
			failure();
			return
		};

		function fail(event) {
			// TODO: error message.
			failure();
		};

		let req = new XMLHttpRequest();
		req.addEventListener("load", function(event) {
			database = new Database(event.target.response);
			fetchedDB = true;
			success(database);
		});
		req.addEventListener("error", fail);
		req.addEventListener("abort", fail);
		req.open("GET", "/ref/search.db");
		req.responseType = "arraybuffer";
		req.send();
		return;
	};
	success(database);
	return;
}

function generateLink(item, devhub) {
	let split = item.name.indexOf(".");
	let parent = "";
	let member = item.name;
	if (split > 0) {
		parent = item.name.slice(0,split);
		member = item.name.slice(split+1);
	};
	if (devhub) {
		switch (item.dbType) {
		case "class":
			return "/class/"+encodeURI(member);
		case "enum":
			return "/enum/"+encodeURI(member);
		case "enumitem":
			return "/enum/"+encodeURI(parent);
		case "type":
			return "/datatype/"+encodeURI(member);
		case "property":
		case "function":
		case "event":
		case "callback":
			return "/"+item.dbType+"/"+encodeURI(parent)+"/"+encodeURI(member);
		};
		return "";
	} else {
		switch (item.dbType) {
		case "class":
			return "/ref/class/"+encodeURI(member)+".html";
		case "enum":
			return "/ref/enum/"+encodeURI(member)+".html";
		case "enumitem":
			return "/ref/enum/"+encodeURI(parent)+".html#member-"+encodeURI(member);
		case "type":
			return "/ref/type/"+encodeURI(member)+".html";
		case "property":
		case "function":
		case "event":
		case "callback":
			return "/ref/class/"+encodeURI(parent)+".html#member-"+encodeURI(member);
		};
		return "/ref";
	};
};

function generateIcon(item) {
	let data = item.icon;
	if (data === null) {
		return null;
	};
	let icon = document.createElement("span");
	icon.classList.add("icon");
	icon.classList.add(data.class);
	if (data.index) {
		icon.style = "--icon-index: " + data.index;
	};
	return icon
};

function sortResults(a,b) {
	if (a[1].deprecated === b[1].deprecated) {
		if (a[0][0] === b[0][0]) {
			return b[0][1] - a[0][1]
		};
		return (a[0][0] && !b[0][0]) ? 1 : -1;
	};
	return (a[1].deprecated && !b[1].deprecated) ? 1 : -1;
};

document.addEventListener("DOMContentLoaded", function() {
	let search = document.getElementById("search");
	if (search === null) {
		return;
	};
	let main = document.querySelector("main");
	if (main === null) {
		return;
	};

	search.insertAdjacentHTML("afterbegin",
		'<form id="search-form">' +
			'<input type="text" id="search-input" placeholder="Search...">' +
		'</form>'
	);
	main.insertAdjacentHTML("beforebegin", '<section id="search-results" style="display: none;"></section>');

	let searchForm = document.getElementById("search-form");
	if (searchForm === null) {
		return;
	};
	let searchInput = document.getElementById("search-input");
	if (searchInput === null) {
		return;
	};
	let searchResults = document.getElementById("search-results");
	if (searchResults === null) {
		return;
	};

	let firstResult = null;
	function renderResults(results) {
		while (searchResults.lastChild) {
			searchResults.removeChild(searchResults.lastChild);
		};
		if (!results) {
			main.style.display = "";
			searchResults.style.display = "none";
			return;
		};

		results.sort(sortResults);

		main.style.display = "none";
		searchResults.style.display = "";
		let list = document.createElement("ul");
		searchResults.appendChild(list);

		// Limit number of results.
		var max = results.length > 20 ? 20 : results.length;
		firstResult = null;
		for (let i = 0; i < max; i++) {
			let result = results[i];
			let item = document.createElement("li");
			if (result[1].deprecated) {
				item.classList.add("api-deprecated");
			};
			{
				let link = document.createElement("a");
				link.classList.add("element-link");
				link.href = generateLink(result[1]);
				if (firstResult === null) {
					firstResult = link.href;
				};
				let icon = generateIcon(result[1]);
				if (icon !== null) {
					link.appendChild(icon);
				};
				link.innerHTML += result[0][2];
				item.appendChild(link);
			};
			{
				let u = generateLink(result[1], true);
				if (u !== "") {
					let link = document.createElement("a");
					link.classList.add("devhub-link");
					link.href = devhubBaseURL + u;
					let icon = document.createElement("span");
					icon.classList.add("icon");
					icon.classList.add("devhub-icon");
					link.appendChild(icon);
					link.appendChild(document.createTextNode("On DevHub"));
					item.appendChild(link);
				};
			};
			list.appendChild(item);
		};
	};

	let asyncMatcher = null;
	function doSearch(query, render) {
		if (!render) {
			render = renderResults;
		};
		if (query.length === 0) {
			render(null);
			return;
		};

		getDatabase(
			function(database) {
				if (asyncMatcher !== null) {
					asyncMatcher.cancel();
					asyncMatcher = null;
				};
				asyncMatcher = new fts_fuzzy_match_async(query, database, render);
				asyncMatcher.start();
			},
			function() {
				// TODO: error message.
				console.log("DB FAIL");
			}
		);
	};

	// Show results as user types.
	let timer;
	searchInput.addEventListener("input", function() {
		timer && clearTimeout(timer);
		timer = window.setTimeout(function() {
			doSearch(searchInput.value);
		}, 200);
	});

	// Go to the first result when the user presses enter.
	searchForm.addEventListener("submit", function(event) {
		event.preventDefault();
		if (firstResult !== null) {
			var parseURL = document.createElement('a');
			parseURL.href = firstResult;
			if (document.location.pathname == parseURL.pathname) {
				renderResults(null);
				if (parseURL.hash.length > 0) {
					document.location.hash = parseURL.hash;
				}
			} else {
				document.location = firstResult;
			}
		};
	});

	if (searchInput.value.length > 0) {
		doSearch(searchInput.value);
	} else {
		// Try reading URL query.
		let params = new URLSearchParams(document.location.search);
		let q = params.get("q");
		if (q !== null && q !== "") {
			doSearch(q, function(results){
				if (params.get("devhub") === null) {
					renderResults(results);
					return;
				};
				// Automatically redirect to devhub.
				results.sort(sortResults);
				let first = results[0];
				if (first === undefined) {
					return;
				};
				let u = generateLink(first[1], true);
				if (u === "") {
					renderResults(results);
					return;
				};
				document.location = devhubBaseURL + u;
			});
		};
	};
});
