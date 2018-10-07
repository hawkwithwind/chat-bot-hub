(function () {
	app.factory('asyncLoad', asyncLoad);
	asyncLoad.$inject = ['JS_REQUIRES']
	function asyncLoad(jsRequires) {
		function loadSequence() {
            var _args = arguments;
            return {
                deps: ['$ocLazyLoad', '$q',
        			function ($ocLL, $q) {
        			    var promise = $q.when(1);
        			    for (var i = 0, len = _args.length; i < len; i++) {
        			        promise = promiseThen(_args[i]);
        			    }
        			    return promise;

        			    function promiseThen(_arg) {
                            if (typeof _arg == 'function') {
        			            return promise.then(_arg);
        			        } else {
        			            return promise.then(function () {
        			                var nowLoad = requiredData(_arg);
        			                if (!nowLoad)
        			                    return $.error('没有找到指定的模块 [' + _arg + ']');
        			                return $ocLL.load(nowLoad);
        			            });
                            }
        			    }
        			    function requiredData(name) {
        			        if (jsRequires.modules) {
        			            for (var m in jsRequires.modules) {
        			                if (jsRequires.modules[m].name && jsRequires.modules[m].name === name) {
        			                    return jsRequires.modules[m];
                                    }
                                }
                            }
        			        return jsRequires.scripts && jsRequires.scripts[name];
        			    }
        			}]
            };
        }
		return loadSequence;
	}
})();