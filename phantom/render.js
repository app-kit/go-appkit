var webPage = require('webpage');
var system = require('system');
var fs = require("fs");

var args = system.args;
var page = webPage.create();

console.log(args);

if (args.length !== 3) {
	console.log("Two arguments expected");
	phantom.exit(1);
}

var url = args[1];
var filePath = args[2];

console.log("Opening page " + url);
page.open(url, function(status) {
	// setTimeout is a workaround for "unsafe javascript attempt errors"
	// See https://github.com/ariya/phantomjs/issues/12697.
	setTimeout(function() {
		if (status == "fail") {
			console.log("Request failed");
			phantom.exit(1);
		}

		console.log("Saving content to " + filePath);
		fs.write(filePath, page.content, 'w')
	  phantom.exit();
	}, 5);
});

