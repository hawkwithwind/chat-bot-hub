(function(){
    app.factory('timeModel', timeModel);
    function timeModel(){
        var getTime = {
            start:function(type,time){
                var obj = {
                    start_date:"",
                };
                var arr = [];
                if(type == 'daily'){
                    obj.start_date = time;
                }
                if(type == 'weekly'){
                    obj = getWeekDay(time);
                }
                if(type == 'monthly'){
                    var num ='';
                    arr = time.split('-');
                    if([1,3,5,7,8,10,12].indexOf(Number(arr[1])) >= 0 ){
                        //判断是否是大月份
                        arr[2] = '31';
                    }else if(arr[1] == '02'){
                        //判断闰年2月份的天数
                        if(Number(arr[0])%4 == 0){
                            arr[2] = '29';
                        }else{
                            arr[2] = '28';
                        }
                    }else{
                        //小月份
                        arr[2] = '30';
                    }
                    obj.start_date = arr[0] + '-' + arr[1] + '-' + '01';
                }
                return obj
            },
            dayPrev:function(str,time){
                var num;
                var obj = {
                    time:'',
                    type:''
                };
                if(str == 'daily'){
                    num = 1;
                }
                if(str == 'weekly'){
                    num = 7;
                }
                if(str == 'monthly'){
                    var arr = time.split('-');
                    arr[1] = (arr[1] - 1);
                    if(arr[1] == 0){
                        arr[1] = 12
                    }else if(arr[1] == 12){
                        arr[1] = 1
                    }
                    if([1,3,5,7,8,10,12].indexOf(Number(arr[1])) >= 0 ){
                        //判断是否是大月份
                        arr[2] = '31';
                    }else if(arr[1] == '02'){
                        //判断闰年2月份的天数
                        if(Number(arr[0])%4 == 0){
                            arr[2] = '29';
                        }else{
                            arr[2] = '28';
                        }
                    }else{
                        //小月份
                        arr[2] = '30';
                    }
                    num = arr[2];
                }
                obj.time = moment(time).subtract(num, 'day').format('YYYY-MM-DD');
                obj.type = str;
                return obj;
            },
            dayNext:function(str,time){
                var num;
                var obj = {
                    time:'',
                    type:''
                };
                if(str == 'daily'){
                    num = 1;
                }
                if(str == 'weekly'){
                    num = 7;
                }
                if(str == 'monthly'){
                    var arr = time.split('-');
                    if([1,3,5,7,8,10,12].indexOf(Number(arr[1])) >= 0 ){
                        //判断是否是大月份
                        arr[2] = '31';
                    }else if(arr[1] == '02'){
                        //判断闰年2月份的天数
                        if(Number(arr[0])%4 == 0){
                            arr[2] = '29';
                        }else{
                            arr[2] = '28';
                        }
                    }else{
                        //小月份
                        arr[2] = '30';
                    }
                    num = arr[2];
                }
                obj.time = moment(time).add(num, 'day').format('YYYY-MM-DD');
                obj.type = str;
                return obj;
            }
        };
        function getWeekDay(day){
            var obj={};
            var oToday=new Date(day);
            var currentDay=oToday.getDay();
            if(currentDay==0){currentDay=7}
            var mondayTime=oToday.getTime()-(currentDay-1)*24*60*60*1000;
            var sundayTime=oToday.getTime()+(7-currentDay)*24*60*60*1000;
            obj.start_date = timeFormat(new Date(mondayTime).toLocaleDateString());
            return obj
        }
        function timeFormat(time){
            //格式化成需要的时间，非共有方法
            var arra = time.split('/');
            if(arra[1]<10){
                arra[1] = '0' + arra[1]
            }
            if(arra[2]<10){
                arra[2] = '0' +arra[2]
            }
            var str = arra.join('-')
            return str;
        }
        return getTime;
    }
})();
