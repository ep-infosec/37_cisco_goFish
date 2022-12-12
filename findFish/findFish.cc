#include <algorithm>
#include <csignal>
#include <dirent.h>
#include <iostream>
#include <string.h>
#include <thread>

#include "resources/includes/Processor.h"
#include "resources/includes/Calibration.h"

using namespace std;

// Directory to save JSON config video_files to.
#define JSON_DIR "static/video-info/"
#define VIDEO_DIR "static/videos/"

//#define THREADED

void HandleSignal(int);
std::vector<std::string> GetVideosFromDir(std::string, std::vector<std::string>);

int main(int argc, char** argv)
{
    signal(SIGABRT, HandleSignal);
    signal(SIGINT, HandleSignal);
    
    // FIXME: This is very hacky, and should not stay. 
    // See https://github.com/cisco/goFish/projects/1#card-24603535 for possible solution.
    if (argv[1] != NULL) 
    {
        if (std::string(argv[1]) == "TRIANGULATE")
            try 
            {
                Processor p;
                p.TriangulatePoints("calib_config/measure_points.yaml", "stereo_calibration.yaml");
            } 
            catch (const std::exception& e) 
            {
                std::cerr << e.what() << '\n';
            }
        else if (std::string(argv[1]) == "CALIBRATE" )
        try
        {
            Calibration::Input input;
            input.image_size = cv::Size(1920, 1440);

            Calibration calib(input, CalibrationType::STEREO, "stereo_calibration.yaml");
            
            calib.ReadImages(argv[2], argv[3]);
            calib.RunCalibration();
            //calib->GetUndistortedImage();
        }
        catch(const std::exception& e)
        {
            std::cerr << e.what() << '\n';
        }
        

        return 0;
    }

    bool bHasVideos = false;
    do 
    {
        // Check for video files first.
        std::vector<std::string> vid_filters = { ".mp4", ".MP4" };
        auto video_files = GetVideosFromDir(VIDEO_DIR, vid_filters);
        std::sort(video_files.begin(), video_files.end());

        // If there are none, then nothing to do.
        bHasVideos = !video_files.empty();
        if(!bHasVideos) break;

        std::vector<std::string> json_filters = { ".json", ".JSON" };
        auto json_files = GetVideosFromDir(JSON_DIR, json_filters);

        for (size_t i = 0; i < json_files.size(); i++) 
        {
            std::string jf = json_files[i].substr(json_files[i].find("DE_") + 3, json_files[i].length());
            jf = jf.substr(0, jf.find_last_of("."));

            for (auto it = video_files.begin(); it != video_files.end(); ++it)
                if ((*it).find(jf) != std::string::npos) 
                {
                    video_files.erase(it);
                    break;
                }
        }
        std::sort(video_files.begin(), video_files.end());

        if(argv[1] == NULL)
        {
        #ifdef THREADED
            std::vector<std::thread> threads;
            threads.resize(video_files.size());
            for (size_t i = 0; i < video_files.size() / 2; i++)
            {
                std::cout << "!!! Creating Thread: " << i << " !!!\n";
                auto p = new Processor(video_files[i], video_files[(i + 1) % video_files.size()]);
                threads[i] = std::thread(&Processor::ProcessVideos, p);
            }

            for(auto& thread : threads)
                if(thread.joinable()) thread.join();

            // TODO: Should check to make sure processing was successful.
            if(video_files.size() > 1)
                for(std::string video : video_files)
                    std::remove(video.c_str());
        #else
            for (size_t i = 0; i < video_files.size() / 2; i += 2)
                try
                {
                    Processor p(video_files[i], video_files[(i + 1) % video_files.size()]);
                    p.ProcessVideos();

                    if(p.Success)
                    {
                        std::remove(video_files[i].c_str());
                        std::remove(video_files[(i + 1) % video_files.size()].c_str());
                    }
                }
                catch(const std::exception& e)
                {
                    std::cerr << e.what() << '\n';
                }
        #endif
        }

    } while (bHasVideos);

    return 0;
}

void HandleSignal(int signal)
{
    std::cout << "\r=== Got signal: " << signal << " ===" << endl;
    std::cout << "  > Terminating..." << endl;
    exit(0);
}

std::vector<std::string> GetVideosFromDir(std::string dir, std::vector<std::string> filters)
{
    std::vector<std::string> video_files;
    DIR* dp;
    struct dirent* d;
    if ((dp = opendir(dir.c_str())) != NULL)
    {
        while ((d = readdir(dp)) != NULL)
            for (auto filter : filters)
                if ((strcmp(string(d->d_name).c_str(), ".") != false && strcmp(string(d->d_name).c_str(), "..") != false) && string(d->d_name).find(filter) != string::npos)
                    video_files.push_back(dir + d->d_name);
        closedir(dp);
    }
    return video_files;
}