import Foundation
import XCTest
import RudderStackAnalytics

enum EventValidation {
    case track(name: String, properties: [String: Any] = [:])
    case identify(userId: String, traits: [String: Any] = [:])
    case screen(screenName: String, properties: [String: Any] = [:])
    case group(groupId: String, traits: [String: Any] = [:])
}

final class EventValidationPlugin: Plugin {
    var pluginType: PluginType = .onProcess
    var analytics: Analytics?

    private let lock = NSLock()
    private var received: [Event] = []
    private var validationIndex: Int = 0

    func setup(analytics: Analytics) {
        self.analytics = analytics
    }

    func teardown() {
        analytics = nil
    }

    func intercept(event: Event) -> Event? {
        lock.lock()
        received.append(event)
        lock.unlock()
        return nil
    }

    func validateCount(_ expected: Int, timeout: TimeInterval = 5.0, file: StaticString = #file, line: UInt = #line) {
        let deadline = Date().addingTimeInterval(timeout)
        while currentCount() < expected {
            if Date() >= deadline {
                XCTFail("Timeout waiting for events. Expected \(expected), got \(currentCount()) after \(timeout)s", file: file, line: line)
                return
            }
            Thread.sleep(forTimeInterval: 0.05)
        }
        let actual = currentCount()
        if actual > expected {
            XCTFail("Received more events than expected. Expected \(expected), got \(actual)", file: file, line: line)
        }
    }

    func validateNext(_ expected: EventValidation, file: StaticString = #file, line: UInt = #line) {
        lock.lock()
        let index = validationIndex
        let event = index < received.count ? received[index] : nil
        validationIndex += 1
        lock.unlock()

        guard let event = event else {
            XCTFail("No event at index \(index)", file: file, line: line)
            return
        }
        validate(event: event, against: expected, file: file, line: line)
    }

    private func currentCount() -> Int {
        lock.lock()
        defer { lock.unlock() }
        return received.count
    }

    private func validate(event: Event, against expected: EventValidation, file: StaticString, line: UInt) {
        switch expected {
        case .track(let name, let properties):
            guard let track = event as? TrackEvent else {
                XCTFail("Expected TrackEvent, got \(type(of: event))", file: file, line: line)
                return
            }
            XCTAssertEqual(track.event, name, "Track event name mismatch", file: file, line: line)
            assertJSONEqual(track.properties, properties, label: "Track properties", file: file, line: line)

        case .identify(let userId, let traits):
            guard let identify = event as? IdentifyEvent else {
                XCTFail("Expected IdentifyEvent, got \(type(of: event))", file: file, line: line)
                return
            }
            XCTAssertEqual(identify.userId, userId, "Identify userId mismatch", file: file, line: line)
            // Traits live under context["traits"] after the SDK processes the event.
            let contextTraits = identify.context?["traits"]?.value
            assertJSONEqual(contextTraits, traits, label: "Identify traits", file: file, line: line)

        case .screen(let screenName, let properties):
            guard let screen = event as? ScreenEvent else {
                XCTFail("Expected ScreenEvent, got \(type(of: event))", file: file, line: line)
                return
            }
            XCTAssertEqual(screen.event, screenName, "Screen event name mismatch", file: file, line: line)
            assertJSONEqual(screen.properties, properties, label: "Screen properties", file: file, line: line)

        case .group(let groupId, let traits):
            guard let group = event as? GroupEvent else {
                XCTFail("Expected GroupEvent, got \(type(of: event))", file: file, line: line)
                return
            }
            XCTAssertEqual(group.groupId, groupId, "Group groupId mismatch", file: file, line: line)
            assertJSONEqual(group.traits, traits, label: "Group traits", file: file, line: line)
        }
    }

    private func assertJSONEqual(_ received: Any?, _ expected: [String: Any], label: String, file: StaticString, line: UInt) {
        let receivedJSON = canonicalJSON(fromCodable: received)
        let expectedJSON = canonicalJSON(fromDictionary: expected)
        XCTAssertEqual(receivedJSON, expectedJSON, "\(label) mismatch", file: file, line: line)
    }

    private func canonicalJSON(fromCodable value: Any?) -> String {
        guard let value = value else { return "{}" }
        // If it's a CodableCollection, encode via JSONEncoder; otherwise assume [String: Any].
        if let collection = value as? CodableCollection {
            guard let data = try? JSONEncoder().encode(collection),
                  let obj = try? JSONSerialization.jsonObject(with: data),
                  let reencoded = try? JSONSerialization.data(withJSONObject: obj, options: [.sortedKeys]) else {
                return "<encode-failed>"
            }
            return String(data: reencoded, encoding: .utf8) ?? "<utf8-failed>"
        }
        if let dict = value as? [String: Any] {
            return canonicalJSON(fromDictionary: dict)
        }
        return "<unexpected-type: \(type(of: value))>"
    }

    private func canonicalJSON(fromDictionary dict: [String: Any]) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: dict, options: [.sortedKeys]),
              let str = String(data: data, encoding: .utf8) else {
            return "<serialize-failed>"
        }
        return str
    }
}
