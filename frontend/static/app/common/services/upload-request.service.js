(function() {
	app

		.factory('uploadRequest',  uploadRequest);
		uploadRequest.$inject = ['$q', '$http', 'toastr'];

		function uploadRequest ($q, $http, toastr) {
			return function (url, formData, config) {
				var defered = $q.defer();
				config = config || {};
				if (!url) {
					toastr.error('上次url不能为空');
					return false;
				}
				if (!formData) {
					toastr.error('上次数据不能为空');
					return false;
				}

				$http({
				　　	method: 'POST',
				　　 url: url,
				  	data: config.data || {},
				  	headers: {
				    	'Content-Type': undefined
				 	},
				  	transformRequest: function(data) {
				    	return formData;
				  	}
			  	}).success(function(d) {
				    //请求成功
				    if (d.isSuccess) {
				    	defered.resolve(d);
				    } else {
				    	defered.reject(d);
				    }
			  	}).error(function(err, status) {
				    defered.reject(err);
			  	});
			  	return defered.promise;




				//var xhr = new XMLHttpRequest(),
				// 	defered = $q.defer();
				// xhr.responseType = 'json';
				// xhr.timeout = config.timeout || 5000;
				// xhr.ontimeout = function(){
				// 	xhr.abort();
				// 	toastr.error("网络异常，上传超时");
				// };
				// xhr.onerror = function(){
				// 	toastr.error("未知错误，上传失败");
				// };
				// xhr.open('POST', url);

				// // xhr.upload.onprogress = function(evt){  
				// // 	// 已上传大小
			 // //        var loaded = evt.loaded; 
			 // //        // 总大小
			 // //        var tot = evt.total;
			 // //        // 已经上传的百分比  
			 // //        var per = Math.floor( 100 * loaded / tot);
			 // //        var son =  document.getElementById('son');
			 // //        son.innerHTML = per + "%";  
			 // //        son.style.width = per + "%";  
			 // //    };
				// xhr.onreadystatechange = function(){
				// 	if(xhr.readyState == 4 && xhr.status == 200){
				// 		defered.resolve(xhr.response);
				// 	} else if (xhr.readyState == 4 && xhr.status != 200) {
				// 		defered.reject(xhr.response);
				// 	}
				// };
				//xhr.send(formData);
			};
		}
})();