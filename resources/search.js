"use strict";
{
const devhubBaseURL = "https://developer.roblox.com/api-reference";
const maxResults = 50;

let statusFilter = null;
function initStatusFilters() {
	let securityID = 0;
	let deprecated = true;
	let unbrowsable = true;
	let hidden = true;
	let removed = true;
	window.rbxapiSettings.Listen("SecurityIdentity", function(name, value, initial) {
		securityID = Number(value);
	});
	window.rbxapiSettings.Listen("ShowDeprecated", function(name, value, initial) {
		deprecated = value;
	});
	window.rbxapiSettings.Listen("ShowNotBrowsable", function(name, value, initial) {
		unbrowsable = value;
	});
	window.rbxapiSettings.Listen("ShowHidden", function(name, value, initial) {
		hidden = value;
	});
	window.rbxapiSettings.Listen("ShowRemoved", function(name, value, initial) {
		removed = value;
	});
	statusFilter = function(item) {
		if (item.deprecated && !deprecated) {
			return true;
		};
		if (item.unbrowsable && !unbrowsable) {
			return true;
		};
		if (item.hidden && !hidden) {
			return true;
		};
		if (item.removed && !removed) {
			return true;
		};

		if (securityIdentities && securityContexts && securityID > 0) {
			let ctxIndex = -1;
			let sec = item.security;
			if (sec && typeof(sec) == "string") {
				ctxIndex = securityContexts.indexOf(sec);
			} else if (sec) {
				let r = securityContexts.indexOf(sec.read);
				let w = securityContexts.indexOf(sec.write);
				r = r < 0 ? securityContexts.length-1 : r;
				w = w < 0 ? securityContexts.length-1 : w;
				ctxIndex = r > w ? r : w;
			};
			ctxIndex = ctxIndex < 0 ? securityContexts.length-1 : ctxIndex;
			let idIndex = securityIdentities.indexOf(securityID);
			if (ctxIndex <= idIndex) {
				return true;
			};
		};
		return false;
	};
};

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

function escapeHTML(text) {
	let e = document.createElement("div");
	e.innerText = text;
	return e.innerHTML;
};

// Returns [bool, score, formattedStr]
// bool: true if each character in pattern is found sequentially within str
// score: integer; higher is better match. Value has no intrinsic meaning. Range varies with pattern.
//        Can only compare scores with same search pattern.
// formattedStr: input str with matched characters marked in <b> tags. Delete if unwanted.
function fuzzy_match(pattern, str) {

	// Score consts
	var perfect_bonus = 100;                // bonus for perfect, case-insensitive matches
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

	// Apply bonus for perfect match
	if (pattern.toLowerCase() === str.toLowerCase()) {
		score += perfect_bonus;
	};

	// Finish out formatted string after last pattern matched
	// Build formated string based on matched letters
	var formattedStr = "";
	var lastIdx = 0;
	for (var i = 0; i < matchedIndices.length; ++i) {
		var idx = matchedIndices[i];
		formattedStr += escapeHTML(str.substr(lastIdx, idx - lastIdx)) + "<b>" + escapeHTML(str.charAt(idx)) + "</b>";
		lastIdx = idx + 1;
	}
	formattedStr += escapeHTML(str.substr(lastIdx, str.length - lastIdx));

	var matched = patternIdx == patternLength;
	return [matched, score, formattedStr];
}

// Strictly optional utility to help make using fts_fuzzy_match easier for large data sets
// Uses setTimeout to process matches before a maximum amount of time before sleeping
//
// To use:
//      var asyncMatcher = fts_fuzzy_match(fuzzy_match, "fts", "ForrestTheWoods",
//                                         function(results) { console.log(results); });
//      asyncMatcher.start();
//
function fts_fuzzy_match_async(pattern, database, onComplete) {
	var queryType = null;
	var queryTypeSplit = pattern.indexOf(":");
	if (queryTypeSplit >= 0) {
		queryType = pattern.slice(0, queryTypeSplit).toLowerCase();
		if (!validEntityTypes.has(queryType)) {
			onComplete([]);
			return null;
		};
		pattern = pattern.slice(queryTypeSplit+1);
	};
	var queryPrimarySplit = pattern.indexOf(".");

	var ITEMS_PER_CHECK = 1000;         // performance.now can be very slow depending on platform

	var max_ms_per_frame = 1000.0/30.0; // 30FPS
	var itemIndex = 0;
	var itemCount = database.itemCount;
	var itemOffset = database.STRINGS;
	var results = [];
	var resumeTimeout = null;
	var decoder = new TextDecoder();

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

			let item = database.item(itemIndex);
			itemIndex++;

			let len = database.data.getUint8(itemOffset);
			itemOffset++;
			item.name = decoder.decode(database.data.buffer.slice(itemOffset, itemOffset + len))
			itemOffset += len;

			if (queryType) {
				let type = item.dbType;
				if (queryType === "member") {
					switch (type) {
					case "property":
					case "function":
					case "event":
					case "callback":
						break;
					default:
						continue;
					};
				} else if (type !== queryType) {
					continue;
				};
			};

			if (statusFilter !== null && statusFilter(item)) {
				continue;
			};

			let prefix = "";
			let suffix = item.name;
			if (queryPrimarySplit < 0) {
				let split = item.name.indexOf(".");
				if (split >= 0) {
					prefix = item.name.slice(0,split+1);
					suffix = item.name.slice(split+1);
				};
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

	return {
		// Abort current process
		cancel: function() {
			if (resumeTimeout !== null)
				clearTimeout(resumeTimeout);
		},
		// Must be called to start matching.
		// I tried to make asyncMatcher auto-start via "var resumeTimeout = step();"
		// However setTimout behaving in an unexpected fashion as onComplete insisted on triggering twice.
		start: function() {
			step();
		},
		// Process full list. Blocks script execution until complete
		flush: function() {
			max_ms_per_frame = Infinity;
			step();
		},
	};

};
/* END fts_fuzzy_match.js */


function getbit(p, a) {
	return (p>>a)&1 === 1;
};
function getbits(p, a, b) {
	return (p >> a) & ((1<<(b-a)) - 1);
};

function securityString(sec) {
	if (!securityContexts) {
		return "None";
	};
	sec = sec >= securityContexts.length ? 0 : sec;
	return securityContexts[securityContexts.length-1-sec];
};

let validEntityTypes = new Set([
	"class",
	"enum",
	"enumitem",
	"type",
	"member",
	"property",
	"function",
	"event",
	"callback",
])

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
	get unbrowsable() {
		return !!getbit(this.data, 5);
	};
	get uncreatable() {
		if (getbits(this.data, 0, 3) == 0) {
			return !!getbit(this.data, 6);
		};
		return null;
	};
	get hidden() {
		if (getbits(this.data, 0, 3) == 4) {
			return !!getbit(this.data, 6);
		};
		return null;
	};
	get protected() {
		let type = getbits(this.data, 0, 3)
		if (type < 4) {
			return null;
		};
		if (getbits(this.data, 8, 11) > 0) {
			return true;
		};
		return type == 4 && getbits(this.data, 11, 14) > 0
	};
	get security() {
		let type = getbits(this.data, 0, 3);
		if (type < 4) {
			return null;
		};
		if (type == 4) {
			return {
				read:  securityString(getbits(this.data, 8, 11)),
				write: securityString(getbits(this.data, 11, 14)),
			}
		};
		return securityString(getbits(this.data, 8, 11));
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
		this.ICON_SIZE = 1;
		this.ITEM_SIZE = 2;

		this.VERSION      = 0;
		this.ICON_COUNT   = this.VERSION      + 1;
		this.CLASS_OFFSET = this.ICON_COUNT   + 2;
		this.ITEM_COUNT   = this.CLASS_OFFSET + 2;
		this.ICONS        = this.ITEM_COUNT   + 2;
		this.ITEMS        = this.ICONS        + this.ICON_SIZE*this.iconCount;
		this.STRINGS      = this.ITEMS        + this.ITEM_SIZE*this.itemCount;
	};
	get version() {
		return this.data.getUint8(this.VERSION);
	};
	get iconCount() {
		return this.data.getUint16(this.ICON_COUNT, true);
	};
	get itemCount() {
		return this.data.getUint16(this.ITEM_COUNT, true);
	};
	get classOffset() {
		return this.data.getUint16(this.CLASS_OFFSET, true);
	};
	icon(index) {
		index = index % this.iconCount;
		return this.data.getUint8(this.ICONS + this.ICON_SIZE*index);
	};
	itemData(index) {
		index = index % this.itemCount;
		return this.data.getUint16(this.ITEMS + this.ITEM_SIZE*index, true);
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
		let off = this.STRINGS;
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
	item(index, useString) {
		index = index % this.itemCount;
		let item = new DatabaseItem(this.itemData(index))
		if (useString) {
			item.name = this.string(index)
		};
		if (item.dbType === "class") {
			item.iconIndex = this.icon(index - this.classOffset);
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

		let dbPath = document.head.querySelector("meta[name=\"search-db\"]");
		if (dbPath === null) {
			return;
		};
		dbPath = dbPath.content;

		let req = new XMLHttpRequest();
		req.addEventListener("load", function(event) {
			database = new Database(event.target.response);
			fetchedDB = true;
			success(database);
		});
		req.addEventListener("error", fail);
		req.addEventListener("abort", fail);
		req.open("GET", dbPath);
		req.responseType = "arraybuffer";
		req.send();
		return;
	};
	success(database);
	return;
}

function doubleEncode(uri) {
	return encodeURI(encodeURI(uri));
};

let pathSub = null;
function generateLink(item, devhub) {
	if (pathSub === null) {
		let tag = document.head.querySelector("meta[name=\"path-sub\"]");
		if (tag === null) {
			pathSub = "/ref";
		} else {
			pathSub = tag.content;
		};
	};

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
			return "/class/"+doubleEncode(member);
		case "enum":
			return "/enum/"+doubleEncode(member);
		case "enumitem":
			return "/enum/"+doubleEncode(parent);
		case "type":
			return "/datatype/"+doubleEncode(member);
		case "property":
		case "function":
		case "event":
		case "callback":
			return "/"+item.dbType+"/"+doubleEncode(parent)+"/"+doubleEncode(member);
		};
		return "";
	} else {
		switch (item.dbType) {
		case "class":
			return pathSub+"/class/"+doubleEncode(member)+".html";
		case "enum":
			return pathSub+"/enum/"+doubleEncode(member)+".html";
		case "enumitem":
			return pathSub+"/enum/"+doubleEncode(parent)+".html#member-"+doubleEncode(member);
		case "type":
			return pathSub+"/type/"+doubleEncode(member)+".html";
		case "property":
		case "function":
		case "event":
		case "callback":
			return pathSub+"/class/"+doubleEncode(parent)+".html#member-"+doubleEncode(member);
		};
		return pathSub;
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
	// [0][0]: matched bool
	// [0][1]: score   int
	// [0][2]: value   string
	if (a[0][0] === b[0][0]) {
		return b[0][1] - a[0][1]
	};
	return (a[0][0] && !b[0][0]) ? 1 : -1;
};

function initSearch() {
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
			'<input type="text" id="search-input" placeholder="Press S to search...">' +
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
		firstResult = null;
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

		if (results.length === 0) {
			let item = document.createElement("div");
			item.innerText = "No results";
			list.appendChild(item);
			return;
		};

		// Limit number of results.
		var max = results.length > maxResults ? maxResults : results.length;
		for (let i = 0; i < max; i++) {
			let result = results[i];
			let item = document.createElement("li");
			if (result[1].deprecated) {
				item.classList.add("api-deprecated");
			};
			if (result[1].unbrowsable) {
				item.classList.add("api-not-browsable");
			};
			if (result[1].hidden) {
				item.classList.add("api-hidden");
			};
			if (result[1].removed) {
				item.classList.add("api-removed");
			};
			let sec = result[1].security;
			if (sec !== null) {
				if (typeof(sec) === "string") {
					item.classList.add("api-sec-" + sec);
				} else {
					item.classList.add("api-sec-" + sec.read);
					item.classList.add("api-sec-" + sec.write);
				};
			};
			item.title = "score: " + result[0][1];

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
			if (!result[1].removed) {
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
				asyncMatcher = fts_fuzzy_match_async(query, database, render);
				if (asyncMatcher !== null) {
					asyncMatcher.start();
				};
			},
			function() {
				// TODO: error message.
				console.log("DB FAIL");
			}
		);
	};

	// Shortcuts to focus on search bar.
	document.addEventListener("keydown",function(e) {
		if (e.altKey || e.ctrlKey || e.metaKey) {
			return;
		};
		if (e.key === "Escape" && searchInput === document.activeElement) {
			searchInput.blur();
			return;
		};
		if ((e.key === "s" || e.key === "S") && searchInput !== document.activeElement) {
			e.preventDefault();
			searchInput.focus();
			searchInput.select();
			return;
		};
	});

	// Show results as the user types.
	let timer;
	searchInput.addEventListener("input", function() {
		timer && clearTimeout(timer);
		timer = window.setTimeout(function() {
			doSearch(searchInput.value);
		}, 200);
	});

	// Reshow results on focus.
	searchInput.addEventListener("focus", function() {
		if (searchInput.value.length > 0) {
			doSearch(searchInput.value);
		};
	});

	// Hide results when a result on the current page is selected.
	searchResults.addEventListener("click", function(event) {
		let anchor = event.target.closest("a");
		if (anchor === null) {
			return;
		};
		if (document.location.origin == anchor.origin &&
			document.location.pathname == anchor.pathname) {
			renderResults(null);
		};
	});

	// Go to the first result when the user presses enter.
	searchForm.addEventListener("submit", function(event) {
		event.preventDefault();
		if (firstResult !== null) {
			var parseURL = document.createElement('a');
			parseURL.href = firstResult;
			if (document.location.origin == parseURL.origin &&
				document.location.pathname == parseURL.pathname) {
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
				if (u === "" || first[1].removed) {
					renderResults(results);
					return;
				};
				document.location = devhubBaseURL + u;
			});
		};
	};
};

if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", initSearch);
} else {
	initSearch();
};


if (window.rbxapiSettings) {
	initStatusFilters();
} else {
	window.addEventListener("rbxapiSettings", initStatusFilters);
};

};
