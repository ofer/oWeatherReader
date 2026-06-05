import { Injectable, Optional, SkipSelf, InjectionToken, Inject } from '@angular/core';
import { HouseHvacRecommendation, WeatherReport } from './weather-report';
import { DeviceModel } from './device-model';
import { WeatherReportEvent } from './weather-report-event';
import { HttpClient } from '@angular/common/http';
import { Observable, defer, retry, shareReplay, switchMap, timer } from 'rxjs';

export const EVENT_SOURCE_TOKEN = new InjectionToken<typeof EventSource>('EventSource', {
  providedIn: 'root',
  factory: () => EventSource
});

export interface DailyAgg {
  date: string;
  model: string;
  avgTemp: number;
  highTemp: number;
  lowTemp: number;
  avgHighTemp?: number;
  avgLowTemp?: number;
  avgHumidity: number;
  avgHighHumidity?: number;
  avgLowHumidity?: number;
  highHumidity: number;
  lowHumidity: number;
  modelName: string;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  getHistoricDataForDeviceModel(deviceModel: string) : Observable<WeatherReport[]>{
    return this.http.get<WeatherReport[]>(`./reports/${deviceModel}`)
  }

  getDailyAggregates(models: string[], days: number): Observable<DailyAgg[]> {
    const params = new URLSearchParams({ days: String(days) });
    if (models.length > 0) {
      params.set('models', models.join(','));
    }
    return this.http.get<DailyAgg[]>(`./reports/history?${params.toString()}`)
  }

  getModels(): Observable<DeviceModel[]> {
    return this.http.get<DeviceModel[]>('./models');
  }

  getConfig(): Observable<any> {
    return this.http.get<any>('./config');
  }

  getLatestRecommendedReport(): Observable<HouseHvacRecommendation> {
    return this.http.get<HouseHvacRecommendation>('./recommendations/latest');
  }

  latestReportObserver: Observable<WeatherReport>;

  private weatherReportEventsSubject: Observable<WeatherReportEvent>;

  constructor(private http: HttpClient,
              @Optional() @SkipSelf() private parent: ApiService,
              @Optional() @Inject(EVENT_SOURCE_TOKEN) private EventSourceTok: typeof EventSource) {
    if (parent) {
      throw new Error('ApiService cannot be instantiated more than once.');
    }

    this.latestReportObserver = timer(0, 30000).pipe(
      switchMap(() => this.http.get<WeatherReport>('./reports/latest').pipe(
        retry({ delay: 30000 })
      ))
    );

    this.weatherReportEventsSubject = defer(() => {
      const EventSourceCtor = this.EventSourceTok || (typeof EventSource !== 'undefined' ? EventSource : null);
      const eventSource = new EventSourceCtor!('./reports/events');

      return new Observable<WeatherReportEvent>(observer => {
        eventSource.onmessage = (event: MessageEvent) => {
          try {
            const parsed = JSON.parse(event.data) as WeatherReportEvent;
            observer.next(parsed);
          } catch (err) {
            observer.error(err);
          }
        };

        eventSource.onerror = (event: Event) => {
          observer.error(event);
        };

        return () => {
          eventSource.close();
        };
      });
    }).pipe(
      shareReplay({ bufferSize: 1, refCount: true })
    );
  }

  get weatherReportEvents$(): Observable<WeatherReportEvent> {
    return this.weatherReportEventsSubject;
  }
}
