(function () {
  app.factory('URL', URL);
  URL.$inject = ['$window'];
  function URL($window) {
    var result = {
      'login': buildUrl('login'),      
      'bots': buildUrl('bots'),
      'clients': buildUrl('clients'),
      'botaction/failing': buildUrl('botaction/failing'),
      'filters': buildUrl('filters'),
      'consts': buildUrl('consts'),
      'botlogin': buildUrl('botlogin'),
      'botlogout': buildUrl('botlogout'),
      'botaction': buildUrl('botaction'),
      'loginqq': buildUrl('loginqq'),
      'chatusers': buildUrl('chatusers'),
      'chatgroups': buildUrl('chatgroups'),
      'filtertemplatesuites': buildUrl('filtertemplatesuites')
    };
    return result;
    
    function buildUrl(url) {
      var host = '';
      var pathprefix = $window.location.pathname;

      host = $window.location.protocol + "//" + $window.location.host;
      /*
      if($window.location.port != "80" && $window.location.port != "443" && $window.location.port != "") {
	host += ":" + $window.location.port;
      }
      */
      //console.log(host +"," + pathprefix + "," + url);
      return host + pathprefix + url;
    }
  }
})();
