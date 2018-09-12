function toggleList(event) {
	var list = event.target.parentElement.querySelector(".patch-list");
	if (list === null) {
		return;
	};
	if (list.classList.contains("hidden")) {
		list.classList.remove("hidden");
	} else {
		list.classList.add("hidden");
	};
};
function toggleAll(show, scroll) {
	var scrollTo;
	for (item of document.querySelectorAll("#update-list > li .patch-list")) {
		var anchor = item.parentElement.querySelector("a.anchor:target");
		if (anchor !== null) {
			scrollTo = anchor;
		}
		if (show) {
			item.classList.remove("hidden");
		} else {
			if (anchor !== null) {
				item.classList.remove("hidden");
			} else {
				item.classList.add("hidden");
			};
		};
	};
	if (scroll && scrollTo !== undefined) {
		scrollTo.scrollIntoView(true);
	};
};
document.addEventListener("DOMContentLoaded", function(event) {
	var showAll = document.getElementById("show-all");
	if (showAll !== null) {
		showAll.onclick = function(event) {
			toggleAll(event.target.checked, false);
		};
		toggleAll(showAll.checked, true);
	} else {;
		toggleAll(false, true);
	};
	for (item of document.querySelectorAll("#update-list > li .patch-list-toggle")) {
		item.onclick = toggleList;
	};
});
