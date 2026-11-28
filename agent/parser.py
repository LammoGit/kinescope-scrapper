from connection import SiteConnection
import re


class KinescopeParser:
    def search_for_metadata(self, connection, url, agent_name):
        result = connection.site_connect(url)
        print(result[1])
        if result[0] != None:
            script_body = (
                re.search(
                    r'<script[^>]*type=["\']application/ld\+json["\'][^>]*>(.*?)</script>',
                    result[0],
                    re.DOTALL | re.IGNORECASE,
                )
            ).group()
            if script_body:
                video_name = re.search(r'"name.*"', script_body).group()
                video_description = re.search(
                    r'"description"\s*:\s*"(.*?)"\s*,', script_body, re.DOTALL
                ).group()
                thumbnail_url = re.search("thumbnailUrl.*", script_body).group()[16:][
                    :-2
                ]
                m3u8_url = re.search("contentUrl.*", script_body).group()
                if not m3u8_url:
                    m3u8_url = re.search("embedUrl.*", script_body).group()
                    if m3u8_url:
                        m3u8_url = (m3u8_url[:20] + m3u8_url[26:])[
                            12:-1
                        ] + "/media.m3u8"
                    else:
                        print(f"[{agent_name} status]: not_found for {url}")
                        return None
                elif ".svg" in m3u8_url:
                    print(f"[{agent_name} status]: not_found for {url}")
                    return None
                else:
                    m3u8_url = m3u8_url[14:-14] + "/media.m3u8"
                if not m3u8_url:
                    print(f"[{agent_name} status]: not_found for {url}")
                    return None
                else:
                    print(f"[{agent_name} status]: found for {url}")
                    return [m3u8_url, video_name, video_description, thumbnail_url]

    def download_video(self, connection, m3u8_url, dir_name, agent_name):
        catalog = dir_name + "/"
        for quality in ["480", "720", "1080"]:
            video_result = connection.site_connect(
                m3u8_url + "?quality=" + quality + "&type=video"
            )
            audio_result = connection.site_connect(
                m3u8_url + "?quality=" + quality + "&type=audio&lang=und"
            )
            if video_result[0] != None and audio_result[0] != None:
                video_url = re.search("https.*mp4", video_result[0]).group()
                audio_url = re.search("https.*mp4", audio_result[0]).group()
                if video_url != None and audio_url != None:
                    connection.download_media(video_url, catalog + quality, "video.mp4")
                    connection.download_media(audio_url, catalog + quality, "audio.mp4")
        print(
            f"[{agent_name} status]: http://kinescope.io/{dir_name} video was downloaded successfuly."
        )
