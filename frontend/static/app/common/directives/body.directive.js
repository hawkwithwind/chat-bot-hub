(function () {
	app.directive('body', globalBody);
	function globalBody() {

		function checkEdited($input) {
	        $input.val() != "" ? $input.addClass("edited") : $input.removeClass("edited");
	    }

	    $("body").on("keydown", ".form-md-floating-label .form-control", function(t) {
	        checkEdited($(this));
	    }).on("blur", ".form-md-floating-label .form-control", function(t) {
	        checkEdited($(this));
	    }).on("click", ".md-checkbox > label, .md-radio > label", function() {
	        var $input = $(this),
	            $span = $(this).children("span:first-child"),
	        	$clone = $span.clone(true);

	        $span.addClass("inc");
	        $span.before($clone), $("." + $span.attr("class") + ":last", $input).remove();
	    });

	    return {
	    	restrict: 'E'
	    };

	}
})();