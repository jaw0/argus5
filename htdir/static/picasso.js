// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-05 18:19 (EDT)
// Function: picasso paints graphs

var colors = ["#897fd9","#80ab51","#754192","#4bac73",
              "#c97dcb","#3c6e2c","#de77a4","#3fb5b1",
              "#8e396f","#81862f","#435193","#ac8a39",
              "#5890d7","#a95e24","#4d9fc1","#de9258",
              "#3e5676","#754618","#a2aad1","#625427",
              "#927eae","#93a36e","#c891b4","#3e8a78",
              "#724f66","#81aea3","#9d734b","#698599",
              "#b59e7d","#345d5c","#6e7d63","#425439"]

var status_colors = [ null, null, '#88DDFF','#EEEE00', '#FFBB44', '#FF6666', '#BBBBBB', null]

var gcount = 0

function configure_graphs(){

    $('.graph').each(function(){
        var el    = $(this)
        var obj   = el.attr("data-obj")
        var which = el.attr("data-which") || 'samples'
        var darp  = el.attr("data-darp")  // empty == all
        var ctls  = el.attr("data-ctls")

        var wid   = el.width()
        new Graph(el.get(0), obj, which, darp, ctls, wid)
    })
}

var graphd	// for debugging

function Graph(el, obj, which, darp, ctls, width){

    argus.log("new graph " + obj)
    this.el    = el
    this.obj   = obj
    this.ctls  = ctls   // show controls
    this.width = width
    this.pending  = {}
    this.datasets = {}
    this.grctlid  = 'grctl' + gcount
    this.selected = {}
    this.select   = { which: which, darp: {}, obj: {} }
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
            // range
            // darp
            html = '<div class=graphrange><input type=radio name=range value=samples>Day<br>' +
                '<input type=radio name=range value=hours>Month<br>' +
                '<input type=radio name=range value=days>Year<br>' +
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

        // resize labels so they line up nicely
        var maxw = Math.max.apply(Math, $('#' + ctlid + ' .graphlabel').map(function(){ return $(this).width(); }).get());
        argus.log("maxw: " + maxw)
        $('#' + ctlid + ' .graphlabel').width(maxw + 15 + 1)

        // add change/click handlers
        var g = this
        $( '#' + ctlid + ' input[name=range]').change( function(){ g.controlChanged(this) })
        $( '#' + ctlid + ' .graphlabel').click( function(){ g.labelClicked(this) })

    }

    p.controlChanged = function(el){
        var which = $('#' + this.grctlid + ' input[name=range]:checked').val()
        argus.log('control ' + which)
        this.select.which = which
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

    p.Id = function(which, darp, obj){
        return which + " " + darp + " " + obj
    }

    p.statusColor = function(p, color){
        return status_colors[ p.Status ] || color
    }

    p.graphHwab = function(p){
        if( !p || !p.Exp || !p.Delt ) return

        return {Time: p.Time, Min: p.Exp - p.Delt, Max: p.Exp + p.Delt}
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

        // add hwab, 'supplement ' + id
        // or min/max, ave/std?

        if( r.data[ r.data.length - 1].Delt ){
            argus.log("add hwab")
            this.cs.Add( r.data, {
                id:		'supplement ' + id,
                color:		'#ddddff',
                data_func:	this.graphHwab,
                smooth:		1,
                type:		'range',
                shadow:		1
            })
        }else if( r.data[ r.data.length - 1].Max ){
            argus.log("add min/max")
            this.cs.Add( r.data, {
                id:		'supplement ' + id,
                color:		'#ddeeee',
                smooth:		1,
                type:		'range',
                shadow:		1
            })
        }

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

            if( sel.length == 1 )
                this.cs.Show( 'supplement ' + sel[i] )
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

            if( data.length > 1 ){
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
        this.cs.Replace( 'supplement ' + id, data ) // will be ignored if there is no supplement
        // re-render?
        if( this.selected[id] ) this.maybeBuild()
    }


})()
