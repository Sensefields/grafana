'use strict'
function is_grafana_loaded(){
	return window.document.getElementsByTagName("body")[0].className.includes("page-dashboard")
}

function is_kiosk_mode(){
	return window.document.getElementsByTagName("body")[0].className.includes("page-kiosk-mode")
}

function set_kiosk_mode(){
	console.log('Setting kiosk mode!')
	$("body").addClass("page-kiosk-mode")
}

function fix_alert_links() {
	$(".card-item-notice a").each(function(item){
		var alert_href_fix = $(this).attr("href").replace("&edit&tab=alert", "")
		$(this).attr("href", alert_href_fix)
	})  
}

window.fix_interval = undefined


function start_fix_interval(){
	fix_interval = setInterval(function(){
		if(!is_kiosk_mode()){
			set_kiosk_mode()
		}
		fix_alert_links()
	}, 100)
}

window.load_interval = setInterval(function(){
	if (is_grafana_loaded()){
		console.log('Grafana is loaded!')
		parent.postMessage('grafana-loaded','*')
		clearInterval(load_interval)
	}
}, 100)


window.disable_kiosk_mode = function(){
	clearInterval(window.fix_interval)
	$("body").removeClass("page-kiosk-mode")
}


window.addEventListener('message', function(e) {
	var message = e.data;
	if (e.data === "fastrack-fixes"){
		console.log('Apply Fastrack Fixes!')
		start_fix_interval()
	}
});

