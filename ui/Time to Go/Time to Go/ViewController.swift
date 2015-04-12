//
//  ViewController.swift
//  Time to Go
//
//  Created by Brian Seitz on 4/12/15.
//  Copyright (c) 2015 Brian Seitz. All rights reserved.
//

import UIKit

class ViewController: UIViewController, UITableViewDataSource, UITableViewDelegate {

    override func viewDidLoad() {
        super.viewDidLoad()
        // Do any additional setup after loading the view, typically from a nib.
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
        // Dispose of any resources that can be recreated.
    }
    
    /*
curl 'http://ttg.brnstz.com:8000/api/v1/stops?lat=40.729183&lon=-73.95154&&miles=0.5&filter=bus'

    
    private let subways = [
        ["1", "5 minutes"],
        ["2", "7 minutes"],
        ["3", "3 stops away"],
        ["4", "7 stops away"],
        ["5", "approaching"],
    ]
*/
    
    private var results = [
        [
            "route_id": "1",
            "direction_id": 1,
            "lat": 40.730385,
            "lon": -73.951691,
            "stop_name": "GREENPOINT AV/MC GUINESS BL",
            "headsign": "WILLIAMSBURG BRIDGE PLZ",
            
            "scheduled": [
                [
                    "desc": "",
                    "time": "2015-04-12T19:01:49-04:00",
                ]
            ],
            
            "live": [
                [
                    "desc": "approaching",
                    "time": "0001-01-01T00:00:00Z",
                ]
            ]
        ],
        
        [
            "route_id": "2",
            "direction_id": 1,
            "lat": 40.727818,
            "lon": -73.953171,
            "stop_name": "MANHATTAN AV/CALYER ST",
            "headsign": "DOWNTOWN BKLYN FULTON MALL",
            
            "scheduled": [
                [
                    "desc": "",
                    "time": "2015-04-12T19:05:01-04:00",
                ]
            ],
            
            "live": [
                [
                    "desc": "1.4 miles away",
                    "time": "0001-01-01T00:00:00Z",
                ]
            ]
        ]
    ]
    
    // stolen from: http://stackoverflow.com/a/24094777
    /*
    func getJSON(urlToRequest: String) -> NSData{
        var url = NSURL(string: urlToRequest)
        
        return NSData(contentsOfURL: url!)!
    
    func parseJSON(inputData: NSData) -> NSDictionary{
        var error: NSError?
        var d: NSDictionary = NSJSONSerialization.JSONObjectWithData(inputData, options: NSJSONReadingOptions.MutableContainers, error: &error) as NSDictionary
        
        return d
    }
    */

    func tableView(tableView: UITableView, numberOfRowsInSection: Int) -> Int {
       return results.count
    }
    
    func tableView(tableView: UITableView, cellForRowAtIndexPath indexPath: NSIndexPath) -> UITableViewCell {
        var cell = tableView.dequeueReusableCellWithIdentifier("x") as? UITableViewCell
        
        if (cell == nil) {
            cell = UITableViewCell(
                style: UITableViewCellStyle.Default,
                reuseIdentifier: "x")
        }
        var result = results[indexPath.row]
        
        if let liveResults = result["live"] as? NSArray {
            if let firstResult = liveResults[0] as? NSDictionary {
                if let desc = firstResult["desc"] as? String {
                    cell!.textLabel?.text = desc
                    if let route = result["route_id"] as? String {
                        cell!.imageView?.image = UIImage(named: route)
                    }
                }
            }
        }
        
        
        return cell!
    }

    func tableView(tableView: UITableView, didSelectRowAtIndexPath indexPath: NSIndexPath) {
        
    }
}