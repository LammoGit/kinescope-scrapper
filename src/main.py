from cli import CommandLineParser
from connection import SiteConnection
import re


def main():
    option_parser = CommandLineParser()
    options = option_parser.parse()

    connection = SiteConnection(options)
    result = connection.site_connect(options.base_url)
    print(result[1])
    if result[0] != None:
        m3u8_url = re.search("contentUrl.*", result[0]).group()
        if not m3u8_url:
            m3u8_url = re.search("embedUrl.*", result[0]).group()
            if m3u8_url:
                m3u8_url = m3u8_url[12:-1] + "/media.m3u8"
            else:
                print(
                    f"[{options.agent_name} status]: not_found for {options.base_url}"
                )
        else:
            m3u8_url = m3u8_url[14:-14] + "/media.m3u8"
        if not m3u8_url:
            print(f"[{options.agent_name} status]: not_found for {options.base_url}")
        else:
            print(f"[{options.agent_name} status]: found for {options.base_url}")
            catalog = options.base_url[-22:] + "/"
            for quality in ["480", "720", "1080"]:
                video_url = m3u8_url + "?quality=" + quality + "&type=video"
                audio_url = m3u8_url + "?quality=" + quality + "&type=audio&lang=und"
                video_result = connection.site_connect(video_url)
                audio_result = connection.site_connect(audio_url)
                if video_result[0] != None and audio_result[0] != None:
                    video_url = re.search("https.*mp4", video_result[0]).group()
                    audio_url = re.search("https.*mp4", audio_result[0]).group()
                    if video_url != None and audio_url != None:
                        connection.download_media(
                            video_url, catalog + quality, "video.mp4"
                        )
                        connection.download_media(
                            audio_url, catalog + quality, "audio.mp4"
                        )


if __name__ == "__main__":
    main()
