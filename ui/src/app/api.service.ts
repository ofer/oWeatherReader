import { Injectable } from '@angular/core';
import { HouseHvacRecommendation, WeatherReport } from './weather-report';
import { DeviceModel } from './device-model';
import { WeatherReportEvent } from './weather-report-event';
import { HttpClient } from '@angular/common/http';
import { Observable, delay, interval, retry, retryWhen, shareReplay, switchMap, takeWhile, tap, timer } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  getHistoricDataForDeviceModel(deviceModel: string) : Observable<WeatherReport[]>{
    return this.http.get<WeatherReport[]>(`./reports/${deviceModel}`)
  }

  getModels():Observable<DeviceModel[]>{
    return this.http.get<DeviceModel[]>('./models');
  }

  getLatestRecommendedReport(): Observable<HouseHvacRecommendation> {
    return this.http.get<HouseHvacRecommendation>('./recommendations/latest');
  }

  latestReportObserver: Observable<WeatherReport>;

  private weatherReportEventsSubject: Observable<WeatherReportEvent>;

  constructor(private http: HttpClient) {
    this.latestReportObserver = timer(0, 30000).pipe(
      switchMap(() => this.http.get<WeatherReport>('./reports/latest').pipe(
        retry({ delay: 30000 })
      ))
    );

    this.weatherReportEventsSubject = new Observable<WeatherReportEvent>(observer => {
      const eventSource = new EventSource('./reports/events');

      eventSource.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data) as WeatherReportEvent;
          observer.next(parsed);
        } catch (err) {
          observer.error(err);
        }
      };

      eventSource.onerror = (event) => {
        observer.error(event);
      };

      return () => {
        eventSource.close();
      };
    }).pipe(
      shareReplay({ bufferSize: 1, refCount: true })
    );
  }

  get weatherReportEvents$(): Observable<WeatherReportEvent> {
    return this.weatherReportEventsSubject;
  }
}
