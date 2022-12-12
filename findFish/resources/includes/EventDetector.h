/// \author Tomas Rigaux
/// \date June 18, 2019
///
/// All kinds of events are defined here, as they are small enough that they
/// can easily fit in here while not big enougbto justify having their own
/// respsective files. These events all assume that they happen within a range
/// of time in a video (as opposed to one-shot event), and thus exist between
/// a range of frames defined in a video.

#pragma once

#include <opencv2/opencv.hpp>
#include <opencv2/objdetect.hpp>
#include <opencv2/imgcodecs.hpp>

#include <map>
#include <string>
#include <mutex>

class JSON;

/// Abstract Base class for defining an event.
class EventBuilder
{
 public:
  /// Default constructor.
  EventBuilder();

  /// Default destructor.
  virtual ~EventBuilder();

  /// Check frame for some sort of event.
  /// \param[in, out] The current frame.
  virtual void CheckFrame(cv::Mat& frame, int&) = 0;

  /// Defines the starting frame of the event.
  /// \param[in, out] frame The frame number that marks the start of the event.
  virtual void StartEvent(int& frame)  = 0;

  /// Defines the ending frame of the event.
  /// \param[in, out] frame The frame number that marks the end of the event.
  virtual void EndEvent(int& frame) = 0;

  /// Returns the event as a JSON object.
  /// \return The event formatted into a JSON object.
  const JSON GetAsJSON();

  /// Get the range of the event frames.
  /// \returns The start and end frames as a pair.
  std::pair<int, int> GetRange() const;

 protected:
  std::mutex _mutex;
  cv::Mat _frame;
  int _start_frame, _end_frame;
  std::unique_ptr<JSON> _json_object;
};

/// Defines an event which attempts to detect a QR code from a frame.
class QREvent : public EventBuilder
{
 public:
  /// Constructs the event around a specific frame.
  QREvent();

  /// Default destructor.
  virtual ~QREvent() {};
  
  /// Check frame for some sort of event.
  /// \param[in, out] frame The frame in which to check.
  /// \param[in, out] The current frame.
  virtual void CheckFrame(cv::Mat&, int&) override;

  /// Denotes the start of the event, and begins checking for a QR code.
  /// param[in, out] frame The starting frame of the event.
  virtual void StartEvent(int&) override;
  
  /// Denotes the end of the event, and stops checking for a QR code.
  /// param[in, out] frame The ending frame of the event.
  virtual void EndEvent(int&) override;

  /// Returns whether or not a QR code was found.
  /// \return If the QR code was detected or not.
  const bool DetectedQR() const;
 
 private:
  /// Parses the QR code URL for a Geo URI.
  /// \return All the key-value pairs found in the URL.
  std::map<std::string, std::string> GetGeoURIValues(std::string& uri) const;

};

/// Defines an event in which there was activity of some sort.
class ActivityEvent : public EventBuilder
{
 public:
  /// Constructs an event with a unique ID, which extends in a range from start
  /// to end.
  /// \param[in] id The unique ID of the event.
  /// \param[in, out] start The starting frame of the event.
  /// \param[in, out] end The ending frame of the event.
  ActivityEvent(int id, int start, int end);

  /// Default destructor for the class.
  virtual ~ActivityEvent() {};

  /// Check frame for some sort of event.
  /// \param[in, out] frame The frame in which to check.
  /// \param[in, out] The current frame.
  virtual void CheckFrame(cv::Mat&, int&) override;

  /// Denotes the start of the event
  /// param[in, out] frame The starting frame of the event.
  virtual void StartEvent(int&) override;

  /// Denotes the end of the event
  /// param[in, out] frame The ending frame of the event.
  virtual void EndEvent(int&) override;

  /// Checks whether or not the event is still happening.
  /// \return The running state of the event.
  bool IsActive() const;

 private:
   int id_;
};