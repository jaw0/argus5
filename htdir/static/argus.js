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

var MONTH = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
var DAY = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]

var webtime = 0
var jsondata

console.log("argusjs")

function argus_onload(){
    argus.log("argus onload")
}

function argus_page(){

    argus.log("page loaded")

    build_page()
}

function build_page(){

    spinner_on()

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
    spinner_off()
}

function build_page_ok(d){

    argus.log("json ok")
    argus.log(" json data: " + d)
    spinner_off()

    process_meta(d)
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

    configure_topnav_buttons()
}

// ****************************************************************

function configure_topnav_buttons(){

    //argus.log("conf tnav buttons " + $('#overridebutton'))

}

// ****************************************************************
function override_init(){
    $(document).keydown(function(e) {
        // 27 = escape
        if (e.which == 27) {
            override_dismiss();
        }
    });
}

function override_show(){

    argus.log("override")
    override_init()

    // reset form
    $('.overridedivinner input[name=text]').val( "" );

    $('.overridedivinner').hide();
    $('#overridediv').fadeIn();
    $('.overridedivinner').slideDown();
}
function override_dismiss(){
    $('.overridedivinner').slideUp();
    $('#overridediv').fadeOut();
}
function override_save(){

    var args = { obj: objname }

    args.text    = $('.overridedivinner input[name=text]').val();
    args.mode    = $('.overridedivinner select[name=mode]').val();
    args.expires = $('.overridedivinner select[name=expires]').val();
    args.xtok    = token
    argus.log("save override " + args )

    override_dismiss()
    spinner_on()

    $.ajax({
        type:	    'POST',
        url:	    '/api/override',
        data:       args,
        dataType:   'json',
        timeout:    5000,
        success:    override_success,
        error:      override_error,
    });
}

function override_remove(){

    var args = { obj: objname, remove: 1 }
    args.xtok    = token
    spinner_on()

    $.ajax({
        type:  	    'POST',
        url:	    '/api/override',
        data:       args,
        dataType:   'json',
        timeout:    5000,
        success:    override_success,
        error:      override_error,
    });
}

function override_success(r){
    // r = results from server
    // handle error

    process_meta(r)

    if( r.override ){
        jsondata.mon.override = convert_data(r.override)
    }else{
        jsondata.mon.override = null
    }

    spinner_off()
}
function override_error(r, err){
    spinner_off()
    argus.log("override save error: " + err)
}


// ****************************************************************

function annotate_edit(){
    $('#notesdpy').slideUp();
    $('#notesform').slideDown()
}

function annotate_cancel(){
    $('#notesform').slideUp()
    $('#notesdpy').slideDown();
}

function annotate_save(){

    var args = { obj: objname }

    args.text    = $('#notesform textarea').val();
    args.xtok    = token
    argus.log("save notes " + args )

    $('#notesform').hide()
    spinner_on()

    $.ajax({
        type:	    'POST',
        url:	    '/api/annotate',
        data:       args,
        dataType:   'json',
        timeout:    5000,
        success:    annotate_success,
        error:      annotate_error,
    });
}

function annotate_success(r){
    // r = results from server
    // handle error

    process_meta(r)

    if( r.annotation ){
        jsondata.mon.annotation = convert_data(r.annotation)
    }else{
        jsondata.mon.annotation = null
    }

    $('#notesdpy').slideDown()
    spinner_off()
}

function annotate_error(r, err){
    spinner_off()
    $('#notesdpy').slideDown()
    argus.log("annotate save error: " + err)
}



// ****************************************************************

function checknow(){

    argus.log("check now")

    var args = { obj: objname }

    args.xtok    = token

    $('body').hide()
    spinner_on()

    $.ajax({
        type:	    'POST',
        url:	    '/api/checknow',
        data:       args,
        dataType:   'json',
        timeout:    5000,
        success:    checknow_success,
        error:      checknow_error,
    });
}

function checknow_success(r){

    process_meta(r)
    $('body').show()
    spinner_off()
    // RSN - refetch page json
}

function checknow_error(r, err){

    $('body').show()
    spinner_off()
}




// ****************************************************************

function copy_data(d){

}

function process_meta(d){

    // errors
    // redirect
    // siren


    if( d.unacked ){
        $('#notifiesicon').addClass('redbounce').removeClass('fa-envelope-o').addClass('fa-envelope-open-o')
    }else{
        $('#notifiesicon').removeClass('redbounce').removeClass('fa-envelope-open-o').addClass('fa-envelope-o')
    }

    if( d.hasErrors ){
        $('#haserrorsicon').removeClass('fa-info-circle').addClass('fa-warning')
        $('#haserrorsicon').removeClass('warning').addClass('redbounce')
    }else if( d.hasWarns ){
        $('#haserrorsicon').removeClass('fa-info-circle').addClass('fa-warning')
        $('#haserrorsicon').removeClass('redbounce').addClass('major-f')
    }else{
        $('#haserrorsicon').removeClass('fa-warning').addClass('fa-info-circle')
        $('#haserrorsicon').removeClass('redbounce').removeClass('major-f')
    }
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

    var d = new Date(nano / 1000000)

    var td = DAY[d.getDay()] + " " + d.getDate() + " " + MONTH[d.getMonth()]
    var tt = number_2digits(d.getHours()) + ":" + number_2digits(d.getMinutes()) + ":" + number_2digits(d.getSeconds())
    return td + " " + tt + " " + d.getFullYear()
}
function date_short(nano){
    var d = new Date(nano / 1000000)

    var td = d.getDate() + "/" + MONTH[d.getMonth()]
    var tt = number_2digits(d.getHours()) + ":" + number_2digits(d.getMinutes()) + ":" + number_2digits(d.getSeconds())
    return td + " " + tt
}

function spinner_on(){
    $('#spinnericon').show()
}

function spinner_off(){
    $('#spinnericon').hide()
}
