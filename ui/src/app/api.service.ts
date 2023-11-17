import { Injectable } from '@angular/core';
import { WeatherReport } from './weather-report';
import { DeviceModel } from './device-model';
import { HttpClient } from '@angular/common/http';
import { Observable, interval, switchMap, timer } from 'rxjs';

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

  latestReportObserver: Observable<WeatherReport>;

  constructor(private http: HttpClient) {
    this.latestReportObserver = timer(0,30000).pipe(
      switchMap(() => this.http.get<WeatherReport>('./reports/latest'))
    );
  }
}
