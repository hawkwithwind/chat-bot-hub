(function () {
	"use strict";

	app.factory('requestAnimationFrame', requestAnimationFrame);

	requestAnimationFrame.$inject = ['$window']
	function requestAnimationFrame(window) {
		var requestAnimFrame = (function(){
		  	return  window.requestAnimationFrame       ||
		          	window.webkitRequestAnimationFrame ||
		          	window.mozRequestAnimationFrame    ||
		          	function( callback ){
		            	window.setTimeout(callback, 1000 / 60);
		          	};
		})();

		return requestAnimFrame;
	}

})();