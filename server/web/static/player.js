if (mpegts.getFeatureList().mseLivePlayback) {
	const player = mpegts.createPlayer({
		type: 'mpegts', // or 'flv' if transmuxed flv
		isLive: true,
		url: 'http://localhost:8080/stream/adult-swim.ts'
	});
	player.attachMediaElement(document.getElementById('videoElement'));
	player.load();
	player.play();
}

