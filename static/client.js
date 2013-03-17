var CONFIG = { debug: false
, nick: "#"   // set in onConnect
, id: null    // set in onConnect
, last_message_time: 0
, focus: true //event listeners bound in onConnect
, unread: 0 //updated in the message-processing loop
};

var uuid = "";
var authNick = "";
var authHash = "";

///////////////////////////////////////////////////////////////////////

var nicks = [];
var isNickListDirty = true;

var originalTitle = "";

//  CUT  ///////////////////////////////////////////////////////////////////
/* This license and copyright apply to all code until the next "CUT"
 * http://github.com/jherdman/javascript-relative-time-helpers/
 * 
 * The MIT License
 * 
 * Copyright (c) 2009 James F. Herdman
 * 
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 * 
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 * 
 * 
 * Returns a description of this past date in relative terms.
 * Takes an optional parameter (default: 0) setting the threshold in ms which
 * is considered "Just now".
 *
 * Examples, where new Date().toString() == "Mon Nov 23 2009 17:36:51 GMT-0500 (EST)":
 *
 * new Date().toRelativeTime()
 * --> 'Just now'
 *
 * new Date("Nov 21, 2009").toRelativeTime()
 * --> '2 days ago'
 *
 * // One second ago
 * new Date("Nov 23 2009 17:36:50 GMT-0500 (EST)").toRelativeTime()
 * --> '1 second ago'
 *
 * // One second ago, now setting a now_threshold to 5 seconds
 * new Date("Nov 23 2009 17:36:50 GMT-0500 (EST)").toRelativeTime(5000)
 * --> 'Just now'
 *
 */
Date.prototype.toRelativeTime = function(now_threshold) {
	var delta = new Date() - this;
	
	now_threshold = parseInt(now_threshold, 10);
	
	if (isNaN(now_threshold)) {
		now_threshold = 0;
	}
	
	if (delta <= now_threshold) {
		return 'Just now';
	}
	
	var units = null;
	var conversions = {
		millisecond: 1, // ms    -> ms
		second: 1000,   // ms    -> sec
		minute: 60,     // sec   -> min
		hour:   60,     // min   -> hour
		day:    24,     // hour  -> day
		month:  30,     // day   -> month (roughly)
		year:   12      // month -> year
	};
	
	for (var key in conversions) {
		if (delta < conversions[key]) {
			break;
		} else {
			units = key; // keeps track of the selected key over the iteration
			delta = delta / conversions[key];
		}
	}
	
	// pluralize a unit when the difference is greater than 1.
	delta = Math.floor(delta);
	if (delta !== 1) { units += "s"; }
	return [delta, units].join(" ");
};

/*
 * Wraps up a common pattern used with this plugin whereby you take a String
 * representation of a Date, and want back a date object.
 */
Date.fromString = function(str) {
	return new Date(Date.parse(str));
};

//  CUT  ///////////////////////////////////////////////////////////////////


irisFormatter = function() {
	var emots = [
	[';\\-\\)' , 'wink'],
	[';\\)' , 'wink'],
	
	[';\\-P' , 'tongueout'],
	[';P' , 'tongueout'],
	
	['\\*JOKINGLY\\*' , 'jokingly'],
	
	[':\'\\(' , 'crying'],
	[':\\(\\(' , 'crying'],
	
	['\\*KISSED\\*' , 'kissed'],
	
	[':\\-\\*' , 'kiss'],
	[':\\*' , 'kiss'],
	
	[':\\-\\[' , 'embarassed'],
	
	['O:\\-\\)' , 'angel'],
	['O:\\)' , 'angel'],
	
	[':\\-X' , 'silent'],
	[':X' , 'silent'],
	
	[':\\-\\$' , 'confused'],
	[':\\$' , 'confused'],
	[':\\-S' , 'confused'],
	[':S' , 'confused'],
	
	[':o' , 'angry'],
	
	[':D' , 'laughing'],
	[':\\-D' , 'laughing'],
	[':\\)\\)' , 'laughing'],
	[':\\-\\)\\)' , 'laughing'],
	
	[':\\-\\/' , 'pensive'],
	[':\\\\' , 'pensive'],
	[':\\-\\\\' , 'pensive'],
	
	[':O' , 'shocked'],
	[':\\-O' , 'shocked'],
	
	['B\\)' , 'cool'],
	['B\\-\\)' , 'cool'],
	['8\\)' , 'cool'],
	['8\\-\\)' , 'cool'],
	
	['\\[:\\-\\}' , 'headphone'],
	['\\[:\\}' , 'headphone'],
	['\\[:\\)' , 'headphone'],
	['\\[:\\-\\)' , 'headphone'],
	
	['\\*TIRED\\*' , 'yawning'],
	
	[':\\-\\!' , 'sick'],
	[':\\!' , 'sick'],
	
	['\\*STOP\\*' , 'stop'],
	
	['\\*KISSING\\*' , 'kissing'],
	[':\\*\\*' , 'kissing'],
	
	['\\>:\\)' , 'devil'],
	['\\>:\\-\\)' , 'devil'],
	['\\>:\\-\\>' , 'devil'],
	['>:\\->' , 'devil'],
	
	['\\@\\}\\-\\>\\-\\-' , 'rose'],
	['\\@\\>\\-\\>\\-\\-' , 'rose'],
	['\\@\\>\\-\\>\\-' , 'rose'],
	['\\@\\}\\-\\>\\-' , 'rose'],
	
	['\\@\\=' , 'bomb'],
	
	['\\*THUMBS UP\\*' , 'thumbsup'],
	
	['\\*DRINK\\*' , 'drink'],
	
	['\\*IN LOVE\\*' , 'inlove'],
	
	[':\\-\\)' , 'happy'],
	[':\\)' , 'happy'],
	['\\=\\)'  , 'happy'],
	
	[':\\-\\(' , 'sad'],
	[':\\(' , 'sad']
	];
	
	var qtags = [
	['\\*', 'strong'],
	['\\?\\?', 'cite'],
	['\\+', 'ins'],  //fixed
	['~', 'sub'],
	['\\^', 'sup'], // me
	['@', 'code']
	];
	
	this.reps = [];
	
	for (var i=0;i<emots.length;i++) {
		this.reps.push(['sym_emot_' + i, new RegExp(emots[i][0],'g'), ':emot:' + emots[i][1] + ':']);
	}
	
	this.reps.push(['sym_lt', new RegExp('<','g'), '&lt;']);
	this.reps.push(['sym_gt', new RegExp('>','g'), '&gt;']);
	
	for (var i=0;i<qtags.length;i++) {
		ttag = qtags[i][0]; htag = qtags[i][1];
		this.reps.push(['sh_basic_' + i, new RegExp(ttag+'\\b(.+?)\\b'+ttag,'g'), '<'+htag+'>'+'$1'+'</'+htag+'>']);
	};
	
	this.reps.push(['sym_underscores', new RegExp('\\b_(.+?)_\\b','g'), '<em>$1</em>']);
	
	this.reps.push(['sym_dashes', new RegExp('[\s\n]-(.+?)-[\s\n]','g'), '<del>$1</del>']);
	
	this.reps.push(['sh_link1', new RegExp('"\\b(.+?)\\(\\b(.+?)\\b\\)":([^\\s]+)','g'),
	'<a href="$3" target="_blank" title="$2">$1</a> (Click to open)']);
	
	this.reps.push(['sh_link2', new RegExp('"\\b(.+?)\\b":([^\\s]+)','g'),
	'<a href="$2" target="_blank">$1</a> (Click to open)']);
	/*
	 *	this.reps.push(['sh_youtube', new RegExp('\\%youtube\\=\\b([.\-]+?)\\b\\%','g'),
	 *			'Youtube video $1 (Added to shared playlist). <script>ytplayer_playlist.push( "$1" );</script>']);
	 */
	/*
	 *	this.reps.push(['sh_img1', new RegExp('!\\b(.+?)\\(\\b(.+?)\\b\\)!','g'),
	 *			'Image: <a href="$1" target="_blank" onclick="return embedImage(this, \'$1\')">[$2] (Click to view</a>']);
	 *	this.reps.push(['sh_img2', new RegExp('!\\b(.+?)\\b!','g'),
	 *			'Image: <a href="$1" target="_blank" onclick="return embedImage(this, \'$1\')">(Click to view)</a>']);
	 *	this.reps.push(['sh_youtube', new RegExp('\\%youtube\\=\\b(.+?)\\b\\%','g'),
	 *			'Youtube: <a href="http://www.youtube.com/watch?v=$1" target="_blank" onclick="return embedYoutube(this, \'$1\')">$1 (Click to watch)</a>']);
	 */
	
	this.reps.push(['lh_emots', new RegExp(':emot:(.+?):','g'), '<div class="emot sprite-$1">:emot:$1:</div>']);
	
	var emotPatterns = [];
	
	var metachars = /[[\]{}()*+?.\\|^$\-,&#\s]/g;
}

irisFormatter.prototype.format = function (s) {
	var r = s;
	
	if (r.length > 3) {
		var l = r.length;
		
		if ((r.substr(0,1) == '$') && (r.substr(l-1,1) == '$')) {
			return '<img src="http://chart.apis.google.com/chart?cht=tx&chf=bg,s,FFFFFF00&chl=' + encodeURIComponent(r.substr(1,l-2)) + '" alt="$' + r.substr(1,l-2) + '$" onload="scrollDown()" />';
		}
	}
	
	// Wooooohoooo!
	for (var i=0;i<this.reps.length;i++) {
		r = r.replace(this.reps[i][1],this.reps[i][2]);
	}
	
	return r;
}


var colorAssigner = function() {
	this.colors = ['#ff9900', '#e9e9e9', '#3399cc', '#666699', '#669999', '#990033', '#999966', '#cc9966', '#666633', '#99ccff', '#cccc99', '#ffcc66', '#ff6699', '#ff0066', '#cc3399', '#cccccc', '#cccc99', '#ff66cc', '#999933', '#3399cc'];
	this.nicklookups = [];
	this.colorIndex = 0;
}

colorAssigner.prototype.getColor = function(nick) {
	if (this.nicklookups[nick]) {
		return this.nicklookups[nick];
	} else {
		color = this.colors[this.colorIndex];
		this.colorIndex = (this.colorIndex + 1) % this.colors.length;
		this.nicklookups[nick] = color;
		return color;
	}
}

var formattr = new irisFormatter();
var assignr = new colorAssigner();


//updates the users link to reflect the number of active users
function updateUsersLink ( ) {
	var t = nicks.length.toString() + " user";
	if (nicks.length != 1) t += "s";
	$("#usersLink").text(t);
}

//handles another person joining chat
function userJoin(nick, timestamp) {
	//put it in the stream
	//addMessage(nick, "joined", timestamp, "join");
	$("#last").text(nick + " <<");
	isNickListDirty = true;
}

//handles someone leaving
function userPart(nick, timestamp) {
	//put it in the stream
	$("#last").text(nick + " >>");
	isNickListDirty = true;
}

// utility functions

util = {
	urlRE: /https?:\/\/([-\w\.]+)+(:\d+)?(\/([^\s]*(\?\S+)?)?)?/g, 
	
	//  html sanitizer 
	toStaticHTML: function(inputHtml) {
		inputHtml = inputHtml.toString();
		return inputHtml.replace(/&/g, "&amp;")
		.replace(/</g, "&lt;")
		.replace(/>/g, "&gt;");
	}, 
	
	//pads n with zeros on the left,
	//digits is minimum length of output
	//zeroPad(3, 5); returns "005"
	//zeroPad(2, 500); returns "500"
	zeroPad: function (digits, n) {
		n = n.toString();
		while (n.length < digits) 
			n = '0' + n;
		return n;
	},
	
	//it is almost 8 o'clock PM here
	//timeString(new Date); returns "19:49"
	timeString: function (date) {
		var minutes = date.getMinutes().toString();
		var hours = date.getHours().toString();
		return this.zeroPad(2, hours) + ":" + this.zeroPad(2, minutes);
	},
	
	//does the argument only contain whitespace?
	isBlank: function(text) {
		var blank = /^\s*$/;
		return (text.match(blank) !== null);
	}
};

//used to keep the most recent messages visible
function scrollDown (threshold) {
	//window.scrollBy(0, 100000000000000000);
	offset = threshold || 200;
	if (document.body.clientHeight + document.body.scrollTop + offset > document.body.scrollHeight)
		document.body.scrollTop = document.body.scrollHeight;
	$("#entry").focus();
}

function updateUptime () {
	if (starttime) {
		$("#uptime").text(starttime.toRelativeTime());
	}
}

//inserts an event into the stream for display
//the event may be a msg, join or part type
//from is the user, text is the body and time is the timestamp, defaulting to now
//_class is a css class to apply to the message, usefull for system events

lastSender = "";
//var seq = 0;
function addMessage (from, text, time, _class) {
	if (text === null)
		return;
	
	if (time == null) {
		// if the time is null or undefined, use the current time.
		time = new Date();
	} else if ((time instanceof Date) === false) {
		// if it's a timestamp, interpret it
		time = new Date(time);
	}
	
	//every message you see is actually a table with 3 cols:
	//  the time,
	//  the person who caused the event,
	//  and the content
	var messageElement = $(document.createElement("div"));
	
	messageElement.addClass("message");
	if (_class)
		messageElement.addClass(_class);
	
	// If the current user said this, add a special css class
	var nick_re = new RegExp(CONFIG.nick);
	if (nick_re.exec(text) || from == "sys")
		messageElement.addClass("personal");
	
	// replace URLs with links
	text = text.replace(util.urlRE, '"$&":$&');
	//seq++;
	//from = '' + seq + '. ' + from;
	text = formattr.format(text);
	
	color = assignr.getColor(from );
	
	var content = ''
	+ ((lastSender != from) ?
	'  <div class="msg-info" style="color: ' + color + '"><span class="nick">' + util.toStaticHTML(from) + '</span>' +
	'  <span class="date">' + util.timeString(time) + '</span></div>' 
	: '')
	+ '  <span class="msg-text" style="color: ' + color + '">' + text  + '</span>'
	+ ''
	;
	
	lastSender = from;
	messageElement.html(content);
	
	//the log is the stream that we view
	$("#log").append(messageElement);
	
	//always view the most recent message when it is added
	scrollDown();
}

//process updates if we have any

function processEvent(evt) {
	switch (evt.Code) {
		case "uuid-return":
			
			uuid = evt.Payload;
			// Authenticate now
			$.ajax({ cache: false
				, type: "GET"
				, url: "/chat/auth/"
				, dataType: "json"
				, data: { nick: authNick, uuid: uuid, auth: authHash }
			});
			break;
			
		case "join":
			userJoin(evt.Payload, evt.Timestamp);
			break;
		case "part":
			userPart(evt.Payload, evt.Timestamp);
			break;
		case "msg":
			addMessage(evt.Origin, evt.Payload, evt.Timestamp);
			break;
		default:
			console.log("Unknown respose received");
			break;
	}
}

function longPoll (data) {
	if (transmission_errors > 2) {
		showConnect();
		return;
	}
	
	//process any updates we may have
	//data will be null on the first call of longPoll
	if (data && data.messages) {
		for (var i = 0; i < data.messages.length; i++) {
			var message = data.messages[i];
			
			//track oldest message so we only request newer messages from server
			if (message.timestamp > CONFIG.last_message_time)
				CONFIG.last_message_time = message.timestamp;
			
			//dispatch new messages to their appropriate handlers
			switch (message.type) {
				case "msg":
					if(!CONFIG.focus){
						CONFIG.unread++;
					}
					addMessage(message.nick, message.text, message.timestamp);
					break;
					
				case "announcement":
					addMessage("**system announcement**", message.text, message.timestamp);
					break;
					
				case "join":
					userJoin(message.nick, message.timestamp);
					break;
					
				case "part":
					userPart(message.nick, message.timestamp);
					break;
			}
		}
		//update the document title to include unread message count if blurred
		updateTitle();
		
		//only after the first request for messages do we want to show who is here
		if (first_poll) {
			first_poll = false;
			who();
		}
	}
	
	//make another request
	$.ajax({ cache: false
	, type: "GET"
	, url: "/recv"
	, dataType: "json"
	, data: { since: CONFIG.last_message_time, id: CONFIG.id }
	, error: function () {
		addMessage("", "*** Long poll error. trying again...", new Date(), "error");
	transmission_errors += 1;
	//don't flood the servers on error, wait 10 seconds before retrying
	setTimeout(longPoll, 10*1000);
	}
	, success: function (data) {
		transmission_errors = 0;
		//if everything went well, begin another request immediately
		//the server will take a long time to respond
		//how long? well, it will wait until there is another message
		//and then it will return it to us and close the connection.
		//since the connection is closed when we get data, we longPoll again
		longPoll(data);
	}
	});
}

//submit a new message to the server
function send(msg) {
	if (CONFIG.debug === false) {
		if (msg[0] != "/") {
			jQuery.get("/chat/send/", {uuid: uuid, payload: msg}, function (data) {
					console.log(data);
					processEvent (data);
			}, "json");
		} else {
			payload = msg.substr(1);
			jQuery.get("/chat/command/", {uuid: uuid, payload: payload}, function (data) {
					console.log(data);
					if (data.Payload == "true") {
						addMessage(data.Origin, "Your command is completed successfully", data.Timestamp);
					} else {
						addMessage(data.Origin, "Bad command or file name", data.Timestamp);
					}
			}, "json");
		}
	}
}

function getQueryVariable(variable) {
	var query = window.location.hash;
	var vars = query.split("&");
	for (var i=0;i<vars.length;i++) {
		var pair = vars[i].split("=");
		if (decodeURI(pair[0]) == variable) {
			return decodeURI(pair[1]);
		}
	}
	return null;
	//alert('Query Variable ' + variable + ' not found');
}


//transition the page to the loading screen
function showLoad () {
	$("#connect").hide();
	$("#loading").show();
	$("#toolbar").hide();
}

//transition the page to the main chat view, putting the cursor in the textfield
function showChat (nick) {
	$("#toolbar").show();
	$("#entry").focus();
	
	$("#connect").hide();
	$("#loading").hide();
	
	scrollDown();
}

//we want to show a count of unread messages when the window does not have focus
function updateTitle(){
	if (CONFIG.unread) {
		document.title = "(" + CONFIG.unread.toString() + ") " + originalTitle;
	} else {
		document.title = originalTitle;
	}
}

// daemon start time
var starttime;

//handle the server's response to our nickname and join request
function onConnect (session) {
	if (session.error) {
		alert("error connecting: " + session.error);
		showConnect();
		return;
	}
	
	CONFIG.nick = session.nick;
	CONFIG.id   = session.id;
	starttime   = new Date(session.starttime);
	updateUptime();
	
	//update the UI to show the chat
	showChat(CONFIG.nick);
	
	//listen for browser events so we know to update the document title
	$(window).bind("blur", function() {
		CONFIG.focus = false;
		updateTitle();
	});
	
	$(window).bind("focus", function() {
		CONFIG.focus = true;
		CONFIG.unread = 0;
		updateTitle();
	});
}

//add a list of present chat members to the stream
function outputUsers () {
	$.ajax({ cache: false
				, type: "GET"
				, url: "/chat/userslist/"
				, dataType: "json"
				, data: { uuid: uuid }
				, success: function (data) {
					console.log(data);
					nick_string = "";
					for (i = 0; i < data.Sessions.length; ++i) {
						nick_string += data.Sessions[i].Nick + " (Joined: " + new Date(data.Sessions[i].Joined * 1000).toRelativeTime() + " ago) | ";
					}
					addMessage("users:", nick_string, new Date(), "notice");
				}
			});
	return false;
}

$(document).ready(function() {
	
	//submit new messages when the user hits enter if the message isnt blank
	$("#entry").keypress(function (e) {
		if (e.keyCode != 13 /* Return */) return;
						 var msg = $("#entry").attr("value").replace("\n", "");
		if (!util.isBlank(msg)) send(msg);
		$("#entry").attr("value", ""); // clear the entry field.
	});
	
	$("#usersLink").click(outputUsers);
	
	authNick = getQueryVariable("nick") || "Anonymous";
	authHash = getQueryVariable("auth") || "-";
	
	window.location.hash=""; 
	
	var source = new EventSource('/chat/events/');
	
	// Create a callback for when a new message is received.
	source.onmessage = function(e) {
		// Append the `data` attribute of the message to the DOM.
		var evt = JSON.parse(e.data);
		console.log(evt);
		processEvent (evt);
	};
	
	if (CONFIG.debug) {
		scrollDown();
		return;
	}
	
	
	originalTitle = document.title;
	
});

