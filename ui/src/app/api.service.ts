import { Injectable } from '@angular/core';
import { HouseHvacRecommendation, WeatherReport } from './weather-report';
import { DeviceModel } from './device-model';
import { HttpClient } from '@angular/common/http';
import { Observable, delay, interval, retry, retryWhen, switchMap, takeWhile, tap, timer } from 'rxjs';

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

  constructor(private http: HttpClient) {
    this.latestReportObserver = timer(0, 30000).pipe(
      switchMap(() => this.http.get<WeatherReport>('./reports/latest').pipe(
        retry({ delay: 30000 })
      ))
    );
  }
}
