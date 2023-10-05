import { Injectable } from '@angular/core';
import { WeatherReport } from './weather-report';
import { HttpClient } from '@angular/common/http';
import { Observable, interval, switchMap } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  getHistoricDataForDeviceModel(deviceModel: string) : Observable<WeatherReport[]>{
    return this.http.get<WeatherReport[]>(`/reports/${deviceModel}`)
  }

  latestReportObserver: Observable<WeatherReport>;

  constructor(private http: HttpClient) {
    this.latestReportObserver = interval(30000).pipe(
      switchMap(() => this.http.get<WeatherReport>('/reports/latest'))
    );
  }
}
