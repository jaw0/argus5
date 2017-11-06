// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-05 18:19 (EDT)
// Function: lay paint on canvas. make pretty graphs.

// Downward the various goddess took her flight,
// And drew a thousand colors from the light;
//   -- Virgil, Aeneid
var colors = ["#897fd9","#80ab51","#754192","#4bac73",
              "#c97dcb","#3c6e2c","#de77a4","#3fb5b1",
              "#8e396f","#81862f","#435193","#ac8a39",
              "#5890d7","#a95e24","#4d9fc1","#de9258",
              "#3e5676","#754618","#a2aad1","#625427",
              "#927eae","#93a36e","#c891b4","#3e8a78",
              "#724f66","#81aea3","#9d734b","#698599",
              "#b59e7d","#345d5c","#6e7d63","#425439"]

// What is your favorite color?
// Blue.  No yel--  Auuuuuuuugh!
//   -- Monty Python, Holy Grail

var status_colors = [ null, null, '#88DDFF','#EEEE00', '#FFBB44', '#FF6666', '#BBBBBB', null]
var supplement_color = '#ddddff'

var gcount = 0

// graphs are just a div with a set of attributes

function configure_graphs(cname){

    $('.' + cname).each(function(){
        var el    = $(this)
        var obj   = el.attr("data-obj")
        var which = el.attr("data-which") || 'samples'
        var darp  = el.attr("data-darp")  // empty == all
        var ctls  = el.attr("data-ctls")  // should we display controls?

        var wid   = el.width()
        new Graph(el.get(0), obj, which, darp, ctls, wid)
    })
}

var graphd	// for debugging

function Graph(el, obj, which, darp, ctls, width){

    argus.log("new graph " + obj)
    this.el       = el
    this.obj      = obj
    this.ctls     = ctls
    this.width    = width
    this.pending  = {}
    this.datasets = {}
    this.grctlid  = 'grctl' + gcount
    this.selected = {}
    this.select   = { which: which, darp: {}, supplement: '', obj: {} }
    if( darp ) this.select.darp[ darp ] = 1

    // info{}, cs, darptags{}

    gcount ++
    this.fetchGraphInfo()

    var g = this
    setInterval( function(){ g.periodicUpdate() }, 300000 )
    graphd = this
    return this
}

(function(){
    var p = Graph.prototype

    p.fetchGraphInfo = function(){

        var g = this
        $.ajax({
            dataType: "json",
            url: '/api/graph',
            data: {obj: this.obj},
            success: function(r){   g.gotGraphInfo(r) },
            error: function(a,b,c){ g.ajaxFail(a,b,c) }
        });
    }

    p.ajaxFail = function(r,err,msg){
        argus.log("error: " + msg)
        this.el.innerHTML = 'ERROR: cannot load graph info: ' + msg
    }

    p.gotGraphInfo = function(r){

        argus.log("graph info")
        if( !r.graph ){
            argus.log("no graph info")
            return
        }

        // Title, YLabel, MyId, List[]{Obj, Label, Hwab, Tags[]
        this.info = r.graph

        // create chart
        this.cs   = new ChartStrip(this.el, {
            title:	 this.info.Title,
            ylabel:	 this.info.YLabel,
            draw_border: 0
        })

        if( ! this.info.List ) return
        
        // gather darp tag list
        // what type of graph?
        // assign colors
        this.learnObjs()
        // create selector - range + darp + objs
        this.createControls()
        // select + fetch
        this.selectAll()
        this.updateSelection()
    }

    p.learnObjs = function(){
        var darp = {}
        var i, j, t
        var L = this.info.List

        this.objs = {}

        for(i=0; i<L.length; i++){
            if( !L[i].Tags ) continue
            for(j=0; j<L[i].Tags.length; j++){
                t = L[i].Tags[j]
                darp[ t ] = 1
                L[i].color = colors[i]
                this.objs[ L[i].Obj ] = L[i]

                // darp="" => select all
                if( !this.darp ) this.select.darp[t] = 1
            }
        }
        this.darptags = Object.keys(darp).sort()
    }

    p.createControls = function(){
        var ctlid = this.grctlid
        var div = document.createElement('div')
        div.className = 'graphcontrols'
        div.id = ctlid
        // insertAfter
        this.el.parentNode.insertBefore(div, this.el.nextSibling)

        var html = ''
        var i, t, L

        if( this.ctls ){
            // range, supplement type, darp
            html = '<div class=graphrange>' +
                this.radioButtons('range', [['samples','Day'], ['hours','Month'], ['days', 'Year']]) +
                '</div>' + "\n"

            html += '<div class=graphsupplement>' +
                this.radioButtons('extra', [['', 'None'], ['hwab', 'Predicted'],
                                            ['minmax', 'Min/Max'], ['stdev', 'Std Dev']]) +
                '</div>' + "\n"

            if( this.darptags.length > 1 ){
                html += '<div class=graphdarp>'

                for(i=0; i<this.darptags.length; i++){
                    html += '<input type=checkbox value="' + t + '">' + t + '<br>'
                }
                html += '</div>' + "\n"
            }
        }
        // labels
        if( this.info.List.length > 1 ){
            L = this.info.List

            html += '<div class=graphlabels>'
            for(i=0; i<L.length; i++){
                html += '<div class="graphlabel" data-idx=' + i + '><i class="fa fa-square" style="color:' +
                    L[i].color + '"></i> ' + L[i].Label + '</div>'
            }
            html += '</div>' + "\n"
        }


        html += '<br style="clear:both;">'
        div.innerHTML = html

        // update range selector
        $('#' + ctlid + ' input[name=range][value='+this.select.which+']').attr('checked', 'checked')
        $('#' + ctlid + ' input[name=extra][value=""]').attr('checked', 'checked')

        // resize labels so they line up nicely
        var maxw = Math.max.apply(Math, $('#' + ctlid + ' .graphlabel').map(function(){ return $(this).width(); }).get());
        $('#' + ctlid + ' .graphlabel').width(maxw + 15 + 1)

        var lht = $('#' + ctlid).height()
        $('#' + ctlid + ' .graphrange').height(lht)

        // add change/click handlers
        var g = this
        $( '#' + ctlid + ' input[name=range]').change( function(){ g.controlChanged(this) })
        $( '#' + ctlid + ' input[name=extra]').change( function(){ g.controlChanged(this) })
        $( '#' + ctlid + ' .graphlabel').click( function(){ g.labelClicked(this) })

        this.updateSupplementDpy()
    }

    // choices: [ [value, label], ...]
    p.radioButtons = function(name, choices){
        var i, html=''

        for(i=0; i<choices.length; i++){
            html += '<span class="' + name + choices[i][0] + '"><input type=radio name="' + name + '" value="' +
                choices[i][0] + '">' + choices[i][1] + '</span><br>'
        }
        return html
    }

    p.controlChanged = function(el){
        var which = $('#' + this.grctlid + ' input[name=range]:checked').val()
        var extra = $('#' + this.grctlid + ' input[name=extra]:checked').val()
        argus.log('control ' + which)
        this.select.which = which
        this.select.supplement = extra
        this.updateSelection()
    }
    p.labelClicked = function(el){
        var idx = $(el).attr('data-idx')
        var obj = this.info.List[idx].Obj

        // toggle
        if( this.select.obj[obj] ){
            delete this.select.obj[obj]
            $(el).find('i').removeClass('fa-square').addClass('fa-square-o')
        }else{
            this.select.obj[obj] = 1
            $(el).find('i').removeClass('fa-square-o').addClass('fa-square')
        }

        argus.log('clicked: ' + obj)
        this.updateSelection()
    }

    p.updateSupplementDpy = function(){
        var sel = Object.keys(this.select.obj)
        var obj = sel[0]
        var L = this.info.List

        if( sel.length != 1 ){
            $('#' + this.grctlid + ' .graphsupplement').hide()
            return
        }

        // hwab enabled?
        var hwab = 0
        for(i=0; i<L.length; i++){
            if( (L[i].Obj == obj) && L[i].Hwab ) hwab = 1
        }

        if( this.select.which == 'samples' ){
            // samples - only choice is none|hwab
            if( !hwab ){
                $('#' + this.grctlid + ' .graphsupplement').hide()
                return
            }
            $('#' + this.grctlid + ' .extraminmax').hide()
            $('#' + this.grctlid + ' .extrastdev').hide()
            if( this.select.supplement != 'hwab' )
                $('#' + this.grctlid + ' input[name=extra][value=""]').attr('checked', 'checked')
        }else{
            $('#' + this.grctlid + ' .extraminmax').show()
            $('#' + this.grctlid + ' .extrastdev').show()
        }

        if( hwab ){
            $('#' + this.grctlid + ' .extrahwab').show()
            if( this.select.supplement == 'hwab' )
                $('#' + this.grctlid + ' input[name=extra][value="hwab"]').attr('checked', 'checked')
        }else{
            $('#' + this.grctlid + ' .extrahwab').hide()

            if( this.select.supplement == 'hwab' )
                $('#' + this.grctlid + ' input[name=extra][value=""]').attr('checked', 'checked')
        }

        $('#' + this.grctlid + ' .graphsupplement').show()
    }

    p.Id = function(which, darp, obj){
        return which + " " + darp + " " + obj
    }

    // Nor long the sun his daily course withheld,
    // But added colors to the world reveal'd:
    // When early Turnus, wak'ning with the light,
    //   -- Virgil, Aeneid
    p.statusColor = function(p, color){
        return status_colors[ p.Status ] || color
    }

    p.graphHwab = function(p){
        if( !p || !p.Exp || !p.Delt ) return

        return {Time: p.Time, Min: p.Exp - p.Delt, Max: p.Exp + p.Delt}
    }
    p.graphStdev = function(p){
        var v = p.Value
        var s = p.Stdev
        return {Time: p.Time, Min: v - 2*s, Max: v + 2*s}
    }

    p.selectAll = function(){
        var i
        var L = this.info.List

        for(i=0; i<L.length; i++){
            this.select.obj[ L[i].Obj ] = 1
        }
    }

    p.updateSelection = function(){
        var i, t, id
        var L = this.info.List
        var which = this.select.which
        var obj, darp

        this.selected = {}

        for(i=0; i<L.length; i++){
            obj = L[i].Obj
            if( ! this.select.obj[ obj ] ) continue
            argus.log("obj " + obj)

            if( ! L[i].Tags ) continue

            for(t=0; t<L[i].Tags.length; t++){
                darp = L[i].Tags[t]
                if( ! this.select.darp[ darp ] ) continue

                // fetch it, if we don't already have it
                id = this.Id(which, darp, obj)
                if( ! this.datasets[id] )
                    this.fetchData(which, darp, obj)
                this.selected[ id ] = 1
            }
        }
        this.updateSupplementDpy()
        this.maybeBuild()
    }

    p.fetchData = function(which, darp, obj){
        var g = this

        this.pending[ this.Id(which, darp, obj) ] = 1

        // fetch the graph data
        $.ajax({
            dataType: "json",
            url: '/api/graphd',
            data: {obj: obj, which: which, tag: darp, width: this.width },
            success: function(r){ g.gotData(which, darp, obj, r)},
            error:   function(r){ g.gotFail(which, darp, obj, r) }
        });
    }

    p.gotData = function(which, darp, obj, r){
        var id = this.Id(which, darp, obj)
        delete this.pending[id]
        this.datasets[id] = { data: r.data, which: which, darp: darp, obj: obj }

        // add to chart
        this.cs.Add( r.data, {
            id:		id,
            color:	this.objs[obj].color,
            color_func: this.statusColor,
            smooth:	1,
            type:	'line',
            shadow:	1
            // ...
        })

        this.cs.Add( r.data, {
            id:		'hwab ' + id,
            color:	supplement_color,
            data_func:	this.graphHwab,
            smooth:	1,
            type:	'range'
        })
        this.cs.Add( r.data, {
            id:		'minmax ' + id,
            color:	supplement_color,
            smooth:	1,
            type:	'range'
        })
        this.cs.Add( r.data, {
            id:		'stdev ' + id,
            color:	supplement_color,
            data_func:	this.graphStdev,
            smooth:	1,
            type:	'range'
        })

        this.maybeBuild()
    }
    p.gotFail = function(which, darp, obj, r){
        var id = this.Id(which, darp, obj)
        delete this.pending[id]

        this.maybeBuild()
    }

    p.maybeBuild = function(){
        if( Object.keys(this.pending).length == 0 ){
            this.build()
        }
    }

    p.build = function(){
        var i

        this.cs.HideAll()

        var sel = Object.keys(this.selected)
        for(i=0; i<sel.length; i++){
            argus.log("selected: " + sel[i] )
            this.cs.Show( sel[i] )

            if( sel.length == 1 && this.select.supplement)
                this.cs.Show( this.select.supplement + ' ' + sel[i] )
        }

        this.cs.Render()
    }

    p.periodicUpdate = function(){
        var i, id, ids, set, data, maxt, pt, dt
        var g = this
        var now = (new Date()).valueOf / 1000

        ids = Object.keys(this.datasets)
        for(i=0; i<ids.length; i++){
            id = ids[i]
            set = this.datasets[id]
            data = set.data
            maxt = 0

            if( data && (data.length > 1) ){
                // predict when more data will be added
                maxt = data[ data.length - 1].Time
                pt   = data[ data.length - 2].Time
                argus.log("maxt " + maxt)
                dt = maxt - pt

                if( maxt + dt > now ) continue
            }

            (function(id){
                $.ajax({
                    dataType: "json",
                    url: '/api/graphd',
                    data: {obj: set.obj, darp: set.darp, which: set.which, since: maxt, width: this.width},
                    success: function(r){ g.gotUpdate(id, r)},
                    error:   function(r){ argus.log("update graph failed") }
                });
            })(id)
        }
    }

    p.gotUpdate = function(id, r){
        if( !r.data ) return
        argus.log("got update " + id + " len: " + r.data.length)
        var set = this.datasets[id]
        var data = set.data
        var i

        // add/trim dataset
        for(i=0; i<r.data.length; i++){
            argus.log("+ " + r.data[i].Time )
            data.push( r.data[i] )
            data.shift()
        }
        // update graph data
        this.cs.Replace( id, data )
        // will be ignored if there is no supplement
        this.cs.Replace( 'hwab ' + id, data )
        this.cs.Replace( 'minmax ' + id, data )
        this.cs.Replace( 'stdev ' + id, data )
        // re-render?
        if( this.selected[id] ) this.maybeBuild()
    }


})()
