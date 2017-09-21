// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-18 21:17 (EDT)
// Function: argus javascript


var argus = {
    log: function(msg){
        if( devmode ){
            console.log(msg)
        }
    },
    status: function(st){
        var names = ["Unknown", "Up", "DOWN/Warning", "DOWN/Minor", "DOWN/Major", "DOWN/Critical", "Override", "Depends"]
        return names[st]
    },
    sev: function(st){
        var names = ["unknown", "clear", "warning", "minor", "major", "critical", "override", "depends"]
        return names[st]
    },
    view_page_url: function(id){ return "/view/page?obj=" + encodeURIComponent(id) },

    


    comma: "happy"
}

var webtime = 0
var jsondata

console.log("argusjs")

function argus_onload(){
    argus.log("argus onload")
}

function argus_page(){

    argus.log("page loaded")
    argus.log("json: " + datasrc )

    build_page()
}

function build_page(){

    // fetch json
    jQuery.ajax({
        dataType: "json",
        url: datasrc,
        data: dataarg,
        success: build_page_ok,
        fail: build_page_fail
    });

}

function build_page_fail(){
    alert("error loading data")
}

function build_page_ok(d){

    argus.log("json ok")
    argus.log(" json data: " + d)

    convert_data(d)

    if( jsondata != null ){
        // copy data into existing vue
        copy_data(d)
        return
    }

    // build + configure vue
    jsondata = d

    var app = new Vue({
        el: '#arguspage',
        data: jsondata
    })

}


function copy_data(d){



}


// convert dates + statuses, expand urls, etc
function convert_data(o){
    var kl = Object.keys(o)
    var i

    for( i in kl ){
        var k = kl[i]
        var c = o[k]

        if( c == null ){
            continue
        }
        if( typeof(c) == "object" ){
            // recurse
            convert_data(c)
        }

        if( typeof(c) == "number" ){
            if( (k == "Status") || (k == "OvStatus") || (k == "Sev") || (k == "status") || (k == "ovstatus") ){
                o[k + "_fmt"] = argus.status(c)
                o[k + "_sev"] = argus.sev(c)
                o[k + "_sevf"] = argus.sev(c) + "-f" // reverse color
            }
            if( c > 1500000000000000000 ){
                o[k + "_fmt"] = date_format(c)
            }
        }
        if( k == "Unique" ){
            o["PageUrl"] = argus.view_page_url(c)
        }


    }
    return o
}


function number_2digits(n) {
    if( n > 9 ){
        return "" + n
    }
    return "0" + n
}

function date_format(nano){
    var month = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
    var day = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]

    var d = new Date(nano / 1000000)

    var td = day[d.getDay()] + " " + d.getDate() + " " + month[d.getMonth()]
    var tt = number_2digits(d.getHours()) + ":" + number_2digits(d.getMinutes()) + ":" + number_2digits(d.getSeconds())
    return td + " " + tt + " " + d.getFullYear()
}
