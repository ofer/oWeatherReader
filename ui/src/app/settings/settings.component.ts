import { Component } from '@angular/core';
import { ApiService } from '../api.service';
import { SettingsService } from '../settings.service';
import { DeviceModel } from '../device-model';
import { MatListOption, MatSelectionListChange } from '@angular/material/list';
import { SelectionModel } from '@angular/cdk/collections';
import { MatGridTileHeaderCssMatStyler } from '@angular/material/grid-list';

export type DeviceListItem = {
  name: string;
  deviceModel: string;
  reportCount: number;
  selected: boolean;
}

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent {

  devices: DeviceListItem[] = [];
  // public get selectedDeviceModels(): string[] | null {
  //   return this.settingsService.getMonitoringDeviceNames();
  // }

  public selectDeviceModel(deviceModelName: string, shouldMonitor: boolean) { 
    this.settingsService.setMonitoringDeviceName(deviceModelName, shouldMonitor);
  }
  onSelectionChange(event: MatSelectionListChange, selectionModel: SelectionModel<MatListOption>) {
    this.devices.forEach(element => {
      this.settingsService.setMonitoringDeviceName(element.deviceModel, false);
    });
    selectionModel.selected.forEach(element => {
      this.settingsService.setMonitoringDeviceName(element.value, true);
    });
  }
  constructor(private api: ApiService, private settingsService: SettingsService) {

    let selectedDevices = settingsService.getMonitoringDeviceNames();
    this.api.getModels().subscribe(deviceModels => {
      deviceModels.forEach(element => {
        this.devices.push({
          name: element.Name,
          deviceModel: element.DeviceModel,
          reportCount: element.ReportCount,
          selected: selectedDevices?.includes(element.DeviceModel) ?? false
        });
      });
    });
  }
}
